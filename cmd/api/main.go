package main

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/BenasB/bx2cloud/internal/api"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TODO: package pb instead of api, move generated files
type apiServer struct {
	api.UnimplementedApiServer
}

func (s *apiServer) Greet(context.Context, *api.GreetingRequest) (*api.Greeting, error) {
	response := &api.Greeting{
		Message:   "Hello gRPC world!",
		GreetedAt: timestamppb.New(time.Now()),
	}

	return response, nil
}

func main() {
	lis, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	apiServer := &apiServer{} // TODO: Move out to newServer()
	api.RegisterApiServer(grpcServer, apiServer)
	grpcServer.Serve(lis)
}
