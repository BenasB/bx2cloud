package main

import (
	"log"
	"net"

	"github.com/BenasB/bx2cloud/internal/api/container"
	"github.com/BenasB/bx2cloud/internal/api/container/images"
	"github.com/BenasB/bx2cloud/internal/api/container/logs"
	"github.com/BenasB/bx2cloud/internal/api/interfaces"
	"github.com/BenasB/bx2cloud/internal/api/introspection"
	"github.com/BenasB/bx2cloud/internal/api/network"
	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/api/subnetwork"
	"github.com/BenasB/bx2cloud/internal/api/subnetwork/ipam"
	"google.golang.org/grpc"
)

func main() {
	address := ":8080" // TODO: Make this configurable
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	ipamRepository := ipam.NewMemoryRepository()

	networkRepository := network.NewMemoryRepository(make([]*interfaces.NetworkModel, 0))
	networkConfigurator, err := network.NewNamespaceConfigurator()
	if err != nil {
		log.Fatalf("Failed to create the network configurator: %v", err)
	}

	subnetworkRepository := subnetwork.NewMemoryRepository(make([]*interfaces.SubnetworkModel, 0))
	subnetworkConfigurator := subnetwork.NewBridgeConfigurator(networkConfigurator.GetNetworkNamespaceName, ipamRepository)

	containerRepository, err := container.NewLibcontainerRepository()
	if err != nil {
		log.Fatalf("Failed to create the container repository: %v", err)
	}

	containerConfigurator := container.NewNamespaceConfigurator(
		networkConfigurator.GetNetworkNamespaceName,
		subnetworkConfigurator.GetBridgeName,
		ipamRepository,
	)
	imagePuller, err := images.NewFlatPuller()
	if err != nil {
		log.Fatalf("Failed to create the image puller: %v", err)
	}

	containerLogger, err := logs.NewFsLogger()
	if err != nil {
		log.Fatalf("Failed to create the container logger: %v", err)
	}

	pb.RegisterNetworkServiceServer(grpcServer, network.NewService(networkRepository, subnetworkRepository, networkConfigurator))
	pb.RegisterSubnetworkServiceServer(grpcServer, subnetwork.NewService(subnetworkRepository, networkRepository, subnetworkConfigurator, ipamRepository))
	pb.RegisterContainerServiceServer(grpcServer, container.NewService(containerRepository, subnetworkRepository, containerConfigurator, imagePuller, ipamRepository, containerLogger))
	pb.RegisterIntrospectionServiceServer(grpcServer, introspection.NewService())

	log.Printf("Starting server on %s", address)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
