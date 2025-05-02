package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/cli/inputs"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.in/yaml.v3"
)

func newVpcWriter() *tabwriter.Writer {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "id\tname\tcidr\n")
	return w
}

func printVpc(w *tabwriter.Writer, vpc *pb.Vpc) {
	fmt.Fprintf(w, "%s\t%s\t%s\n", vpc.Id, vpc.Name, vpc.Cidr)
}

func vpcList(client pb.VpcServiceClient) error {
	stream, err := client.List(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}

	w := newVpcWriter()
	defer w.Flush()
	for {
		vpc, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		printVpc(w, vpc)
	}

	return nil
}

func vpcGet(client pb.VpcServiceClient, identifier string) error {
	delete := func(ctx context.Context, req *pb.VpcIdentificationRequest) (*pb.Vpc, error) {
		return client.Get(ctx, req)
	}
	vpc, err := requestVpcByIdentifier(identifier, delete)
	if err != nil {
		return err
	}

	w := newVpcWriter()
	defer w.Flush()
	printVpc(w, vpc)

	return nil
}

func vpcDelete(client pb.VpcServiceClient, identifier string) error {
	delete := func(ctx context.Context, req *pb.VpcIdentificationRequest) (*emptypb.Empty, error) {
		return client.Delete(ctx, req)
	}
	_, err := requestVpcByIdentifier[*emptypb.Empty](identifier, delete)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully deleted %s\n", identifier)

	return nil
}

func vpcCreate(client pb.VpcServiceClient, yamlBytes []byte) error {
	input := &inputs.VpcCreation{}
	if err := yaml.Unmarshal(yamlBytes, &input); err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	req := &pb.VpcCreationRequest{
		Name: input.Name,
		Cidr: input.Cidr,
	}

	resp, err := client.Create(context.Background(), req)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully created %s\n", resp.Id)

	return nil
}

func requestVpcByIdentifier[T any](identifier string, rpc func(context.Context, *pb.VpcIdentificationRequest) (T, error)) (T, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type result struct {
		resp T
		err  error
	}

	idResult := make(chan result, 1)
	nameResult := make(chan result, 1)

	idRequest := pb.VpcIdentificationRequest{
		Identification: &pb.VpcIdentificationRequest_Id{Id: identifier},
	}

	nameRequest := pb.VpcIdentificationRequest{
		Identification: &pb.VpcIdentificationRequest_Name{Name: identifier},
	}

	go func() {
		resp, err := rpc(ctx, &idRequest)
		idResult <- result{resp, err}
	}()

	go func() {
		resp, err := rpc(ctx, &nameRequest)
		nameResult <- result{resp, err}
	}()

	var lastErr error
	for i := 0; i < 2; i++ {
		select {
		case result := <-idResult:
			if result.err == nil {
				return result.resp, nil
			}
			lastErr = result.err
		case result := <-nameResult:
			if result.err == nil {
				return result.resp, nil
			}
			lastErr = result.err
		}
	}

	var zero T
	return zero, lastErr
}
