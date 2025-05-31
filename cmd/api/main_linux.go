package main

import (
	"log"
	"net"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/network"
	"github.com/BenasB/bx2cloud/internal/api/subnetwork"
	"google.golang.org/grpc"
)

func main() {
	address := "localhost:8080"
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	networkRepository := network.NewMemoryRepository(sampleNetworks)
	networkConfigurator := network.NewNamespaceConfigurator()
	subnetworkRepository := subnetwork.NewMemoryRepository(sampleSubnetworks)

	pb.RegisterNetworkServiceServer(grpcServer, network.NewService(networkRepository, subnetworkRepository, networkConfigurator))
	pb.RegisterSubnetworkServiceServer(grpcServer, subnetwork.NewService(subnetworkRepository, networkRepository))

	log.Printf("Starting server on %s", address)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
