package container

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	runspecs "github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

func (s *service) Exec(stream pb.ContainerService_ExecServer) error {
	first, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to read the stream to retrieve the first stream message: %w", err)
	}
	init := first.GetInitialization()
	if init == nil {
		return fmt.Errorf("first message in the stream is expected to be an initialization message")
	}

	container, err := s.repository.Get(init.Identification.Id)
	if err != nil {
		return fmt.Errorf("failed to retrieve the container for command execution: %w", err)
	}

	term := "xterm"
	if init.Terminal != nil {
		term = *init.Terminal
	}

	if len(init.Args) == 0 {
		init.Args = []string{
			"/bin/sh",
			"-c",
			"[ -x /bin/bash ] && exec /bin/bash || exec /bin/sh",
		}
	}

	spec := &runspecs.Process{
		Args: init.Args,
		Env: []string{
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			fmt.Sprintf("TERM=%s", term),
		},
		ConsoleSize: &runspecs.Box{
			Width:  uint(init.ConsoleWidth),
			Height: uint(init.ConsoleHeight),
		},
	}

	process, err := container.StartAdditionalProcess(spec)
	if err != nil {
		return fmt.Errorf("failed to start an additional process in the container: %w", err)
	}
	pty := process.GetPty()
	defer pty.Close()

	log.Printf("Started an additional process in container %d", init.Identification.Id)

	results := make(chan error, 2)
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := pty.Read(buf)
			if errors.Is(err, unix.EIO) {
				// pty child was closed, which is considered a successfull result
				results <- nil
				return
			}
			if err != nil {
				results <- fmt.Errorf("failed to read master console: %w", err)
				return
			}
			if n > 0 {
				err = stream.Send(&pb.ContainerExecResponse{
					Output: &pb.ContainerExecResponse_Stdout{Stdout: buf[:n]},
				})
				if err != nil {
					results <- fmt.Errorf("failed to send bytes from the master console to the client: %w", err)
					return
				}
			}
		}
	}()

	go func() {
		for {
			req, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				results <- fmt.Errorf("client disconnected: %w", err)
				return
			}
			if err != nil {
				results <- fmt.Errorf("failed to read from the client stream: %w", err)
				return
			}

			if p := req.GetStdin(); p != nil {
				if _, err := pty.Write(p); err != nil {
					results <- fmt.Errorf("failed to write to the master console: %w", err)
					return
				}
			}
		}
	}()

	err = <-results

	if err != nil {
		if stopErr := process.Stop(); stopErr != nil {
			return fmt.Errorf("failed to kill the exec process: %w, when the original error was: %w", stopErr, err)
		}
	}

	exitCode, err := process.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return fmt.Errorf("failed to retrieve the state of the process: %w", err)
		}
	}

	err = stream.Send(&pb.ContainerExecResponse{
		Output: &pb.ContainerExecResponse_ExitCode{
			ExitCode: int32(exitCode),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to send the exit code: %w", err)
	}

	log.Printf("Successfully finished an additional process in container %d", init.Identification.Id)

	return nil
}
