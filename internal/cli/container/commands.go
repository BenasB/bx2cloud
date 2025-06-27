package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"golang.org/x/term"
	"google.golang.org/protobuf/types/known/emptypb"
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

	fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", container.Id, container.Image, container.Status, cidr)
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

func Create(client pb.ContainerServiceClient) error {
	// input := &containerCreation{}
	// if err := yaml.Unmarshal(yamlBytes, &input); err != nil {
	// 	return err
	// }

	// if err := input.Validate(); err != nil {
	// 	return err
	// }

	// TODO: (This PR) Implement a way to read the container creation request from a file or stdin.

	req := &pb.ContainerCreationRequest{
		SubnetworkId: 4,
		Image:        "grafana/grafana",
	}

	resp, err := client.Create(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully created %d\n", resp.Id)

	return nil
}

func Exec(client pb.ContainerServiceClient, id uint32, args []string) error {
	termFd := int(os.Stdin.Fd())

	if !term.IsTerminal(termFd) {
		return fmt.Errorf("standard input must be a terminal for container exec")
	}

	oldState, err := term.MakeRaw(termFd)
	if err != nil {
		return fmt.Errorf("failed to put the terminal into a raw mode: %w", err)
	}
	defer term.Restore(termFd, oldState)

	width, height, err := term.GetSize(termFd)
	if err != nil {
		return fmt.Errorf("failed to retrieve the size of the terminal: %w", err)
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
			os.Stdout.Write(p.Stdout)
		case *pb.ContainerExecResponse_ExitCode:
			exitCode = int(p.ExitCode)
		}
	}

	if err := term.Restore(termFd, oldState); err != nil {
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
