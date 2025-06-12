package network

import (
	"context"
	"fmt"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type service struct {
	pb.UnimplementedNetworkServiceServer
	repository           shared.NetworkRepository
	subnetworkRepository shared.SubnetworkRepository
	configurator         configurator
}

func NewService(repository shared.NetworkRepository, subnetworkRepository shared.SubnetworkRepository, configurator configurator) *service {
	return &service{
		repository:           repository,
		subnetworkRepository: subnetworkRepository,
		configurator:         configurator,
	}
}

func (s *service) Get(ctx context.Context, req *pb.NetworkIdentificationRequest) (*pb.Network, error) {
	return s.repository.Get(req.Id)
}

func (s *service) Delete(ctx context.Context, req *pb.NetworkIdentificationRequest) (*emptypb.Empty, error) {
	subnetworks, errors := s.subnetworkRepository.GetAllByNetworkId(req.Id, ctx)
	select {
	case subnetwork, ok := <-subnetworks:
		if ok {
			// TODO: Move to sentinel errors
			return nil, fmt.Errorf("subnetwork with id %d still depends on the network with id %d", subnetwork.Id, req.Id)
		}
	case err, ok := <-errors:
		if ok {
			return nil, err
		}
	}

	network, err := s.repository.Delete(req.Id)
	if err != nil {
		return nil, err
	}

	if err := s.configurator.unconfigure(network); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *service) Create(ctx context.Context, req *pb.NetworkCreationRequest) (*pb.Network, error) {
	newNetwork := &shared.NetworkModel{
		InternetAccess: req.InternetAccess,
	}

	returnedNetwork, err := s.repository.Add(newNetwork)
	if err != nil {
		return nil, err
	}

	// TODO: eventual consistency mechanism?
	if err := s.configurator.configure(returnedNetwork); err != nil {
		return nil, err
	}

	return returnedNetwork, nil
}

func (s *service) Update(ctx context.Context, req *pb.NetworkUpdateRequest) (*pb.Network, error) {
	network, err := s.repository.Update(req.Identification.Id, func(sn *shared.NetworkModel) {
		sn.InternetAccess = req.Update.InternetAccess
	})

	if err != nil {
		return nil, err
	}

	if err := s.configurator.configure(network); err != nil {
		return nil, err
	}

	return network, nil
}

func (s *service) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Network]) error {
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
		case err, ok := <-errors:
			if ok {
				return err
			}
		}
	}
}
