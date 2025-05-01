package main

import (
	"log"
	"net"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/handlers"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	lis, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	var sampleVpcs = []*pb.Vpc{
		&pb.Vpc{
			Id:        "abc-f12",
			Name:      "first-vpc",
			Cidr:      "10.0.1.0/24",
			CreatedAt: timestamppb.New(time.Now().Add(-time.Hour)),
		},
		&pb.Vpc{
			Id:        "def-h41x21",
			Name:      "second-vpc",
			Cidr:      "10.0.2.0/24",
			CreatedAt: timestamppb.New(time.Now().Add(-time.Minute)),
		},
	}
	pb.RegisterVpcServiceServer(grpcServer, handlers.NewVpcService(sampleVpcs))

	grpcServer.Serve(lis)
}
