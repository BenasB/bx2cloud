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

type NetworkService struct {
	pb.UnimplementedNetworkServiceServer
	networks []*pb.Network
}

func NewNetworkService(networks []*pb.Network) *NetworkService {
	serviceNetworks := make([]*pb.Network, len(networks))
	for i, network := range networks {
		serviceNetworks[i] = proto.Clone(network).(*pb.Network)
	}

	return &NetworkService{
		networks: serviceNetworks,
	}
}

func (s *NetworkService) Get(ctx context.Context, req *pb.NetworkIdentificationRequest) (*pb.Network, error) {
	for _, network := range s.networks {
		if network.Id == req.Id {
			return network, nil
		}
	}

	return nil, fmt.Errorf("could not find network")
}

func (s *NetworkService) Delete(ctx context.Context, req *pb.NetworkIdentificationRequest) (*emptypb.Empty, error) {
	for i, network := range s.networks {
		if network.Id == req.Id {
			s.networks = append(s.networks[:i], s.networks[i+1:]...)
			return &emptypb.Empty{}, nil
		}
	}

	return nil, fmt.Errorf("could not find network")
}

func (s *NetworkService) Create(ctx context.Context, req *pb.NetworkCreationRequest) (*pb.Network, error) {
	newNetwork := &pb.Network{
		Id:             id.NextId("network"),
		InternetAccess: req.InternetAccess,
		CreatedAt:      timestamppb.New(time.Now()),
	}

	s.networks = append(s.networks, newNetwork)

	return newNetwork, nil
}

func (s *NetworkService) Update(ctx context.Context, req *pb.NetworkUpdateRequest) (*pb.Network, error) {
	var network *pb.Network
	for _, n := range s.networks {
		if n.Id == req.Identification.Id {
			network = n
			break
		}
	}

	if network == nil {
		return nil, fmt.Errorf("could not find network")
	}

	network.InternetAccess = req.Update.InternetAccess

	return network, nil
}

func (s *NetworkService) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Network]) error {
	for _, network := range s.networks {
		if err := stream.Send(network); err != nil {
			return err
		}
	}
	return nil
}
