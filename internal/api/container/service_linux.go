package container

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"unsafe"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	runspecs "github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
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

func (s *service) Logs(req *pb.ContainerLogsRequest, stream grpc.ServerStreamingServer[pb.ContainerLogsResponse]) error {
	container, err := s.repository.Get(req.Identification.Id)
	if err != nil {
		return err
	}

	data := container.GetData()

	f, err := s.containerLogger.Get(data.Id)
	if err != nil {
		return fmt.Errorf("failed to open container logs: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 8192)
	for {
		n, err := f.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&pb.ContainerLogsResponse{
			Content: buf[:n],
		}); err != nil {
			return err
		}
	}

	if !req.Follow {
		return nil
	}

	fd, err := unix.InotifyInit()
	if err != nil {
		return fmt.Errorf("failed to initialize log watching: %w", err)
	}
	defer unix.Close(fd)

	// We can't use IN_DELETE_SELF here, because it is not sent when the file is deleted (because inotify monitors inodes)
	wd, err := unix.InotifyAddWatch(fd, f.Name(), unix.IN_MODIFY|unix.IN_ATTRIB)
	if err != nil {
		return fmt.Errorf("failed to watch logs: %w", err)
	}
	defer unix.InotifyRmWatch(fd, uint32(wd))

	for {
		n, err := unix.Read(fd, buf)
		if err != nil {
			return fmt.Errorf("failed to read new changes to logs: %w", err)
		}

		offset := 0
		for offset < n {
			raw := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			mask := raw.Mask
			offset += unix.SizeofInotifyEvent + int(raw.Len)

			if mask&unix.IN_ATTRIB > 0 {
				_, err := os.Stat(f.Name())
				if err != nil {
					return fmt.Errorf("log file no longer exists")
				}
				continue
			}

			currentOffset, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				return fmt.Errorf("failed to determine log reader's offset: %w", err)
			}

			fileInfo, err := f.Stat()
			if err != nil {
				return fmt.Errorf("log file no longer exists")
			}

			// If the container was restarted, the log file was truncated, so reset the reader offset
			if fileInfo.Size() < currentOffset {
				_, err = f.Seek(0, io.SeekStart)
				if err != nil {
					return fmt.Errorf("failed to seek to start after truncation: %w", err)
				}
			}

			for {
				n, err := f.Read(buf)
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
				if err := stream.Send(&pb.ContainerLogsResponse{
					Content: buf[:n],
				}); err != nil {
					return err
				}
			}
		}
	}
}
