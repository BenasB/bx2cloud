package network

import (
	"context"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type NetworkService struct {
	pb.UnimplementedNetworkServiceServer
	repository networkRepository
}

func NewNetworkService(repository networkRepository) *NetworkService {
	return &NetworkService{
		repository: repository,
	}
}

func (s *NetworkService) Get(ctx context.Context, req *pb.NetworkIdentificationRequest) (*pb.Network, error) {
	return s.repository.get(req.Id)
}

func (s *NetworkService) Delete(ctx context.Context, req *pb.NetworkIdentificationRequest) (*emptypb.Empty, error) {
	err := s.repository.delete(req.Id)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *NetworkService) Create(ctx context.Context, req *pb.NetworkCreationRequest) (*pb.Network, error) {
	newNetwork := &pb.Network{
		InternetAccess: req.InternetAccess,
	}

	returnedNetwork, err := s.repository.add(newNetwork)
	if err != nil {
		return nil, err
	}

	return returnedNetwork, nil
}

func (s *NetworkService) Update(ctx context.Context, req *pb.NetworkUpdateRequest) (*pb.Network, error) {
	network, err := s.repository.update(req.Identification.Id, func(sn *pb.Network) {
		sn.InternetAccess = req.Update.InternetAccess
	})

	if err != nil {
		return nil, err
	}

	return network, nil
}

func (s *NetworkService) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Network]) error {
	networks, errors := s.repository.getAll(stream.Context())

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
