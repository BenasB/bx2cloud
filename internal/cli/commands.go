package cli

import (
	"context"
	"fmt"
	"io"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"google.golang.org/protobuf/types/known/emptypb"
)

func vpcList(client pb.VpcServiceClient) error {
	stream, err := client.List(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}

	for {
		vpc, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fmt.Println(vpc)
	}

	return nil
}
