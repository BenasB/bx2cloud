package terraform_test

import (
	"fmt"

	pb "github.com/BenasB/bx2cloud/internal/api"
	provider "github.com/BenasB/bx2cloud/internal/terraform"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	host = "localhost:8080"
)

var (
	providerConfig = "provider \"bx2cloud\" {\n" +
		fmt.Sprintf("host = \"%s\"\n", host) +
		"}\n"
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"bx2cloud": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
	grpcClients = createGrpcClients()
)

func createGrpcClients() *provider.Bx2cloudClients {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.NewClient(host, opts...)
	if err != nil {
		panic(err)
	}

	return &provider.Bx2cloudClients{
		Network:    pb.NewNetworkServiceClient(conn),
		Subnetwork: pb.NewSubnetworkServiceClient(conn),
	}
}
