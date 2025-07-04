package main

import (
	"log"
	"net"

	"github.com/BenasB/bx2cloud/internal/api/container"
	"github.com/BenasB/bx2cloud/internal/api/container/images"
	"github.com/BenasB/bx2cloud/internal/api/network"
	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/api/subnetwork"
	"github.com/BenasB/bx2cloud/internal/api/subnetwork/ipam"
	"google.golang.org/grpc"
)

func main() {
	address := "localhost:8080"
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	ipamRepository := ipam.NewMemoryRepository()

	networkRepository := network.NewMemoryRepository(sampleNetworks)
	networkConfigurator, err := network.NewNamespaceConfigurator()
	if err != nil {
		log.Fatalf("Failed to create the network configurator: %v", err)
	}

	subnetworkRepository := subnetwork.NewMemoryRepository(sampleSubnetworks)
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

	pb.RegisterNetworkServiceServer(grpcServer, network.NewService(networkRepository, subnetworkRepository, networkConfigurator))
	pb.RegisterSubnetworkServiceServer(grpcServer, subnetwork.NewService(subnetworkRepository, networkRepository, subnetworkConfigurator, ipamRepository))
	pb.RegisterContainerServiceServer(grpcServer, container.NewService(containerRepository, subnetworkRepository, containerConfigurator, imagePuller, ipamRepository))

	log.Printf("Starting server on %s", address)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
