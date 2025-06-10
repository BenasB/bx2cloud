package network

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.in/yaml.v3"
)

func newWriter() *tabwriter.Writer {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "id\tinternetAccess\n")
	return w
}

func print(w *tabwriter.Writer, network *pb.Network) {
	fmt.Fprintf(w, "%d\t%t\n", network.Id, network.InternetAccess)
}

func List(client pb.NetworkServiceClient) error {
	stream, err := client.List(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}

	w := newWriter()
	defer w.Flush()
	for {
		network, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		print(w, network)
	}

	return nil
}

func Get(client pb.NetworkServiceClient, id uint32) error {
	network, err := client.Get(context.Background(), &pb.NetworkIdentificationRequest{
		Id: id,
	})
	if err != nil {
		return err
	}

	w := newWriter()
	defer w.Flush()
	print(w, network)

	return nil
}

func Delete(client pb.NetworkServiceClient, id uint32) error {
	_, err := client.Delete(context.Background(), &pb.NetworkIdentificationRequest{
		Id: id,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Successfully deleted %d\n", id)

	return nil
}

func Create(client pb.NetworkServiceClient, yamlBytes []byte) error {
	input := &networkCreation{}
	if err := yaml.Unmarshal(yamlBytes, &input); err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	req := &pb.NetworkCreationRequest{
		InternetAccess: input.InternetAccess,
	}

	resp, err := client.Create(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully created %d\n", resp.Id)

	return nil
}

func Update(client pb.NetworkServiceClient, id uint32, yamlBytes []byte) error {
	input := &networkCreation{}
	if err := yaml.Unmarshal(yamlBytes, &input); err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	req := &pb.NetworkUpdateRequest{
		Identification: &pb.NetworkIdentificationRequest{
			Id: id,
		},
		Update: &pb.NetworkCreationRequest{
			InternetAccess: input.InternetAccess,
		},
	}

	resp, err := client.Update(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully updated %d\n", resp.Id)

	return nil
}
