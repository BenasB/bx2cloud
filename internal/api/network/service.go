package network

import (
	"context"
	"fmt"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type NetworkService struct {
	pb.UnimplementedNetworkServiceServer
	repository           shared.NetworkRepository
	subnetworkRepository shared.SubnetworkRepository
}

func NewNetworkService(repository shared.NetworkRepository, subnetworkRepository shared.SubnetworkRepository) *NetworkService {
	return &NetworkService{
		repository:           repository,
		subnetworkRepository: subnetworkRepository,
	}
}

func (s *NetworkService) Get(ctx context.Context, req *pb.NetworkIdentificationRequest) (*pb.Network, error) {
	return s.repository.Get(req.Id)
}

func (s *NetworkService) Delete(ctx context.Context, req *pb.NetworkIdentificationRequest) (*emptypb.Empty, error) {
	subnetworks, errors := s.subnetworkRepository.GetAllByNetworkId(req.Id, ctx)

	select {
	case _, ok := <-subnetworks:
		if ok {
			// TODO: Move to sentinel errors
			return nil, fmt.Errorf("some subnetworks still depend on the network with id %d", req.Id)
		}
	case err := <-errors:
		return nil, err
	}

	err := s.repository.Delete(req.Id)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *NetworkService) Create(ctx context.Context, req *pb.NetworkCreationRequest) (*pb.Network, error) {
	newNetwork := &pb.Network{
		InternetAccess: req.InternetAccess,
	}

	returnedNetwork, err := s.repository.Add(newNetwork)
	if err != nil {
		return nil, err
	}

	return returnedNetwork, nil
}

func (s *NetworkService) Update(ctx context.Context, req *pb.NetworkUpdateRequest) (*pb.Network, error) {
	network, err := s.repository.Update(req.Identification.Id, func(sn *pb.Network) {
		sn.InternetAccess = req.Update.InternetAccess
	})

	if err != nil {
		return nil, err
	}

	return network, nil
}

func (s *NetworkService) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Network]) error {
	networks, errors := s.repository.GetAll(stream.Context())

	for {
		select {
		case network, ok := <-networks:
			if !ok {
				select {
				case err := <-errors:
					return err
				default:
					return nil
				}
			}
			if err := stream.Send(network); err != nil {
				return err
			}
		case err := <-errors:
			return err
		}
	}
}
