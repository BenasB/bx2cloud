package subnetwork

import (
	"context"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	pb.UnimplementedSubnetworkServiceServer
	repository        shared.SubnetworkRepository
	networkRepository shared.NetworkRepository
}

func NewService(subnetworkRepository shared.SubnetworkRepository, networkRepository shared.NetworkRepository) *Service {
	return &Service{
		repository:        subnetworkRepository,
		networkRepository: networkRepository,
	}
}

func (s *Service) Get(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*shared.SubnetworkModel, error) {
	return s.repository.Get(req.Id)
}

func (s *Service) Delete(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*emptypb.Empty, error) {
	err := s.repository.Delete(req.Id)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) Create(ctx context.Context, req *pb.SubnetworkCreationRequest) (*shared.SubnetworkModel, error) {
	if _, err := s.networkRepository.Get(req.NetworkId); err != nil {
		return nil, err
	}

	newSubnetwork := &shared.SubnetworkModel{
		NetworkId:    req.NetworkId,
		Address:      req.Address,
		PrefixLength: req.PrefixLength,
	}

	returnedSubnetwork, err := s.repository.Add(newSubnetwork)
	if err != nil {
		return nil, err
	}

	return returnedSubnetwork, nil
}

func (s *Service) Update(ctx context.Context, req *pb.SubnetworkUpdateRequest) (*shared.SubnetworkModel, error) {
	subnetwork, err := s.repository.Update(req.Identification.Id, func(sn *shared.SubnetworkModel) {
		sn.Address = req.Update.Address
		sn.PrefixLength = req.Update.PrefixLength
	})

	if err != nil {
		return nil, err
	}

	return subnetwork, nil
}

func (s *Service) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[shared.SubnetworkModel]) error {
	subnetworks, errors := s.repository.GetAll(stream.Context())

	for {
		select {
		case subnetwork, ok := <-subnetworks:
			if !ok {
				select {
				case err := <-errors:
					return err
				default:
					return nil
				}
			}
			if err := stream.Send(subnetwork); err != nil {
				return err
			}
		case err := <-errors:
			return err
		}
	}
}
