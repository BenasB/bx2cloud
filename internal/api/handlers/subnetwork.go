package handlers

import (
	"context"
	"fmt"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/id"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SubnetworkService struct {
	pb.UnimplementedSubnetworkServiceServer
	subnetworks []*pb.Subnetwork
}

func NewSubnetworkService(subnetworks []*pb.Subnetwork) *SubnetworkService {
	serviceSubnetworks := make([]*pb.Subnetwork, len(subnetworks))
	for i, subnetwork := range subnetworks {
		serviceSubnetworks[i] = proto.Clone(subnetwork).(*pb.Subnetwork)
	}

	return &SubnetworkService{
		subnetworks: serviceSubnetworks,
	}
}

func (s *SubnetworkService) Get(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*pb.Subnetwork, error) {
	for _, subnetwork := range s.subnetworks {
		if subnetwork.Id == req.Id {
			return subnetwork, nil
		}
	}

	return nil, fmt.Errorf("could not find subnetwork")
}

func (s *SubnetworkService) Delete(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*emptypb.Empty, error) {
	for i, subnetwork := range s.subnetworks {
		if subnetwork.Id == req.Id {
			s.subnetworks = append(s.subnetworks[:i], s.subnetworks[i+1:]...)
			return &emptypb.Empty{}, nil
		}
	}

	return nil, fmt.Errorf("could not find subnetwork")
}

func (s *SubnetworkService) Create(ctx context.Context, req *pb.SubnetworkCreationRequest) (*pb.Subnetwork, error) {
	newSubnetwork := &pb.Subnetwork{
		Id:           id.NextId("subnetwork"),
		Address:      req.Address,
		PrefixLength: req.PrefixLength,
		CreatedAt:    timestamppb.New(time.Now()),
	}

	s.subnetworks = append(s.subnetworks, newSubnetwork)

	return newSubnetwork, nil
}

func (s *SubnetworkService) Update(ctx context.Context, req *pb.SubnetworkUpdateRequest) (*pb.Subnetwork, error) {
	var subnetwork *pb.Subnetwork
	for _, sn := range s.subnetworks {
		if sn.Id == req.Identification.Id {
			subnetwork = sn
			break
		}
	}

	if subnetwork == nil {
		return nil, fmt.Errorf("could not find subnetwork")
	}

	subnetwork.Address = req.Update.Address
	subnetwork.PrefixLength = req.Update.PrefixLength

	return subnetwork, nil
}

func (s *SubnetworkService) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Subnetwork]) error {
	for _, subnetwork := range s.subnetworks {
		if err := stream.Send(subnetwork); err != nil {
			return err
		}
	}
	return nil
}
