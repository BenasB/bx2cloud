package subnetwork

import (
	"context"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SubnetworkService struct {
	pb.UnimplementedSubnetworkServiceServer
	repository        shared.SubnetworkRepository
	networkRepository shared.NetworkRepository
}

func NewSubnetworkService(subnetworkRepository shared.SubnetworkRepository, networkRepository shared.NetworkRepository) *SubnetworkService {
	return &SubnetworkService{
		repository:        subnetworkRepository,
		networkRepository: networkRepository,
	}
}

func (s *SubnetworkService) Get(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*pb.Subnetwork, error) {
	return s.repository.Get(req.Id)
}

func (s *SubnetworkService) Delete(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*emptypb.Empty, error) {
	err := s.repository.Delete(req.Id)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *SubnetworkService) Create(ctx context.Context, req *pb.SubnetworkCreationRequest) (*pb.Subnetwork, error) {
	if _, err := s.networkRepository.Get(req.NetworkId); err != nil {
		return nil, err
	}

	newSubnetwork := &pb.Subnetwork{
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

func (s *SubnetworkService) Update(ctx context.Context, req *pb.SubnetworkUpdateRequest) (*pb.Subnetwork, error) {
	subnetwork, err := s.repository.Update(req.Identification.Id, func(sn *pb.Subnetwork) {
		sn.Address = req.Update.Address
		sn.PrefixLength = req.Update.PrefixLength
	})

	if err != nil {
		return nil, err
	}

	return subnetwork, nil
}

func (s *SubnetworkService) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Subnetwork]) error {
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
