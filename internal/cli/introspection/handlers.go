package introspection

import (
	"context"
	"fmt"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

var version = "dev"

func Version(client pb.IntrospectionServiceClient) {
	fmt.Printf("CLI version: %s\n", version)

	resp, err := client.Get(context.Background(), &emptypb.Empty{})
	if err != nil {
		fmt.Printf("API version: failed to determine\n")
		return
	}

	fmt.Printf("API version: %s\n", resp.Version)
}
