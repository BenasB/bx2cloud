package network

import (
	"context"
	"fmt"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	pb.UnimplementedNetworkServiceServer
	repository           shared.NetworkRepository
	subnetworkRepository shared.SubnetworkRepository
}

func NewkService(repository shared.NetworkRepository, subnetworkRepository shared.SubnetworkRepository) *Service {
	return &Service{
		repository:           repository,
		subnetworkRepository: subnetworkRepository,
	}
}

func (s *Service) Get(ctx context.Context, req *pb.NetworkIdentificationRequest) (*shared.NetworkModel, error) {
	return s.repository.Get(req.Id)
}

func (s *Service) Delete(ctx context.Context, req *pb.NetworkIdentificationRequest) (*emptypb.Empty, error) {
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

func (s *Service) Create(ctx context.Context, req *pb.NetworkCreationRequest) (*shared.NetworkModel, error) {
	newNetwork := &shared.NetworkModel{
		InternetAccess: req.InternetAccess,
	}

	returnedNetwork, err := s.repository.Add(newNetwork)
	if err != nil {
		return nil, err
	}

	return returnedNetwork, nil
}

func (s *Service) Update(ctx context.Context, req *pb.NetworkUpdateRequest) (*shared.NetworkModel, error) {
	network, err := s.repository.Update(req.Identification.Id, func(sn *shared.NetworkModel) {
		sn.InternetAccess = req.Update.InternetAccess
	})

	if err != nil {
		return nil, err
	}

	return network, nil
}

func (s *Service) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[shared.NetworkModel]) error {
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
