package handlers

import (
	"context"
	"fmt"
	"strings"
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

func (s *GreetService) Greet(ctx context.Context, req *pb.GreetingRequest) (*pb.Greeting, error) {
	var message string
	if req.Name != nil {
		message = fmt.Sprintf("Hello from gRPC world to %s!", *req.Name)
	} else {
		message = "Hello? I don't know your name"
	}

	response := &pb.Greeting{
		Message:   message,
		GreetedAt: timestamppb.New(time.Now()),
	}

	return response, nil
}

func (s *GreetService) ShoutGreet(ctx context.Context, req *pb.GreetingRequest) (*pb.Greeting, error) {
	if req.Name != nil {
		newName := strings.ToUpper(*req.Name)
		req.Name = &newName
	}

	return s.Greet(ctx, req)
}
