package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/opencontainers/runc/libcontainer"
	"golang.org/x/term"
	"google.golang.org/protobuf/types/known/emptypb"
)

func newWriter() *tabwriter.Writer {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "id\timage\tstatus\tip\n")
	return w
}

func print(w *tabwriter.Writer, container *pb.Container) {
	status := libcontainer.Status(container.Status)

	cidr := fmt.Sprintf("%d.%d.%d.%d/%d",
		byte(container.Address>>24),
		byte(container.Address>>16),
		byte(container.Address>>8),
		byte(container.Address),
		container.PrefixLength)

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

func Create(client pb.ContainerServiceClient) error {
	// input := &containerCreation{}
	// if err := yaml.Unmarshal(yamlBytes, &input); err != nil {
	// 	return err
	// }

	// if err := input.Validate(); err != nil {
	// 	return err
	// }

	req := &pb.ContainerCreationRequest{
		// TODO: Init from yaml
	}

	resp, err := client.Create(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully created %d\n", resp.Id)

	return nil
}

func Exec(client pb.ContainerServiceClient, id uint32) error {
	oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	rows, cols, _ := term.GetSize(int(os.Stdin.Fd())) // TODO: terminal size?

	stream, err := client.Exec(context.Background())
	if err != nil {
		return err
	}

	stream.Send(&pb.ContainerExecRequest{
		Input: &pb.ContainerExecRequest_Initialization{
			Initialization: &pb.ContainerExecInitializationRequest{
				Identification: &pb.ContainerIdentificationRequest{
					Id: id,
				},
				ConsoleWidth:  int32(cols),
				ConsoleHeight: int32(rows),
			},
		},
	})

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				break
			}
			if n > 0 {
				stream.Send(&pb.ContainerExecRequest{
					Input: &pb.ContainerExecRequest_Stdin{Stdin: buf[:n]},
				})
			}
		}
	}()

	for {
		resp, err := stream.Recv()
		if err != nil {
			return err
		}
		switch p := resp.Output.(type) {
		case *pb.ContainerExecResponse_Stdout:
			os.Stdout.Write(p.Stdout)
		case *pb.ContainerExecResponse_ExitCode:
			fmt.Printf("\n[Exited with code %d]\n", p.ExitCode)
			break
		}
	}
}
