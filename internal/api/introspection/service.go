package introspection

import (
	"context"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

var version = "dev"

type service struct {
	pb.UnimplementedIntrospectionServiceServer
}

func NewService() *service {
	return &service{}
}

func (s *service) Get(ctx context.Context, req *emptypb.Empty) (*pb.IntrospectionResponse, error) {
	return &pb.IntrospectionResponse{
		Version: version,
	}, nil
}
