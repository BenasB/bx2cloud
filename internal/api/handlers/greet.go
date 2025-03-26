package handlers

import (
	"context"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GreetService struct {
	pb.UnimplementedGreetServiceServer
}

func NewGreetService() *GreetService {
	return &GreetService{}
}

func (s *GreetService) Greet(context.Context, *pb.GreetingRequest) (*pb.Greeting, error) {
	response := &pb.Greeting{
		Message:   "Hello gRPC world!",
		GreetedAt: timestamppb.New(time.Now()),
	}

	return response, nil
}
