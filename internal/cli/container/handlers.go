package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"golang.org/x/term"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.in/yaml.v3"
)

func newWriter() *tabwriter.Writer {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "id\timage\tstatus\tip\n")
	return w
}

func print(w *tabwriter.Writer, container *pb.Container) {
	cidr := fmt.Sprintf("%d.%d.%d.%d/%d",
		byte(container.Address>>24),
		byte(container.Address>>16),
		byte(container.Address>>8),
		byte(container.Address),
		container.PrefixLength)

	status := container.Status
	if container.Status == "running" {
		since := time.Since(container.StartedAt.AsTime())
		status = fmt.Sprintf("%s (%s)", container.Status, since.Round(time.Second))
	}

	fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", container.Id, container.Image, status, cidr)
}

func List(client pb.ContainerServiceClient) error {
	stream, err := client.List(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}

	w := newWriter()
	defer w.Flush()
	for {
		container, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		print(w, container)
	}

	return nil
}

func Get(client pb.ContainerServiceClient, id uint32) error {
	container, err := client.Get(context.Background(), &pb.ContainerIdentificationRequest{
		Id: id,
	})
	if err != nil {
		return err
	}

	w := newWriter()
	defer w.Flush()
	print(w, container)

	return nil
}

func Delete(client pb.ContainerServiceClient, id uint32) error {
	_, err := client.Delete(context.Background(), &pb.ContainerIdentificationRequest{
		Id: id,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Successfully deleted %d\n", id)

	return nil
}

func Create(client pb.ContainerServiceClient, yamlBytes []byte) error {
	input := &containerCreation{}
	if err := yaml.Unmarshal(yamlBytes, &input); err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	req := &pb.ContainerCreationRequest{
		SubnetworkId: input.SubnetworkId,
		Image:        input.Image,
		Entrypoint:   input.Entrypoint,
		Cmd:          input.Cmd,
		Env:          input.Env,
	}

	resp, err := client.Create(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully created %d\n", resp.Id)

	return nil
}

func Exec(client pb.ContainerServiceClient, id uint32, args []string) error {
	inputFd := int(os.Stdin.Fd())

	if !term.IsTerminal(inputFd) {
		return fmt.Errorf("standard input must be a terminal for container exec")
	}

	oldState, err := term.MakeRaw(inputFd)
	if err != nil {
		return fmt.Errorf("failed to put the terminal into raw mode: %w", err)
	}
	defer term.Restore(inputFd, oldState)

	output := os.Stdout
	width, height, err := term.GetSize(inputFd)
	if err != nil {
		width, height, err = term.GetSize(int(output.Fd()))
		if err != nil {
			return fmt.Errorf("failed to retrieve the size of the terminal: %w", err)
		}
	}

	stream, err := client.Exec(context.Background())
	if err != nil {
		return err
	}

	var terminal *string
	if envTerm, ok := os.LookupEnv("TERM"); ok {
		terminal = &envTerm
	}

	stream.Send(&pb.ContainerExecRequest{
		Input: &pb.ContainerExecRequest_Initialization{
			Initialization: &pb.ContainerExecInitializationRequest{
				Identification: &pb.ContainerIdentificationRequest{
					Id: id,
				},
				ConsoleWidth:  int32(width),
				ConsoleHeight: int32(height),
				Terminal:      terminal,
				Args:          args,
			},
		},
	})

	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				err = stream.Send(&pb.ContainerExecRequest{
					Input: &pb.ContainerExecRequest_Stdin{Stdin: buf[:n]},
				})
				if err != nil {
					return
				}
			}
		}
	}()

	var exitCode int
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch p := resp.Output.(type) {
		case *pb.ContainerExecResponse_Stdout:
			output.Write(p.Stdout)
		case *pb.ContainerExecResponse_ExitCode:
			exitCode = int(p.ExitCode)
		}
	}

	if err := term.Restore(inputFd, oldState); err != nil {
		return err
	}

	fmt.Printf("Exited with code %d\n", exitCode)

	return nil
}

func Start(client pb.ContainerServiceClient, id uint32) error {
	resp, err := client.Start(context.Background(), &pb.ContainerIdentificationRequest{
		Id: id,
	})

	if err != nil {
		return err
	}

	fmt.Printf("Container %d is now %q\n", resp.Id, resp.Status)

	return nil
}

func Stop(client pb.ContainerServiceClient, id uint32) error {
	resp, err := client.Stop(context.Background(), &pb.ContainerIdentificationRequest{
		Id: id,
	})

	if err != nil {
		return err
	}

	fmt.Printf("Container %d is now %q\n", resp.Id, resp.Status)

	return nil
}

func Logs(client pb.ContainerServiceClient, id uint32) error {
	req := &pb.ContainerLogsRequest{
		Identification: &pb.ContainerIdentificationRequest{
			Id: id,
		},
		Follow: false, // TODO: (This PR) pass follow from CLI flags
	}

	stream, err := client.Logs(context.Background(), req)
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fmt.Fprint(os.Stdout, string(resp.Content))
	}

	return nil
}
