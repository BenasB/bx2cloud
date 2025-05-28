package subnetwork

import (
	"context"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SubnetworkService struct {
	pb.UnimplementedSubnetworkServiceServer
	repository subnetworkRepository
}

func NewSubnetworkService(repository subnetworkRepository) *SubnetworkService {
	return &SubnetworkService{
		repository: repository,
	}
}

func (s *SubnetworkService) Get(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*pb.Subnetwork, error) {
	return s.repository.get(req.Id)
}

func (s *SubnetworkService) Delete(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*emptypb.Empty, error) {
	err := s.repository.delete(req.Id)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *SubnetworkService) Create(ctx context.Context, req *pb.SubnetworkCreationRequest) (*pb.Subnetwork, error) {
	newSubnetwork := &pb.Subnetwork{
		Address:      req.Address,
		PrefixLength: req.PrefixLength,
	}

	returnedSubnetwork, err := s.repository.add(newSubnetwork)
	if err != nil {
		return nil, err
	}

	return returnedSubnetwork, nil
}

func (s *SubnetworkService) Update(ctx context.Context, req *pb.SubnetworkUpdateRequest) (*pb.Subnetwork, error) {
	subnetwork, err := s.repository.update(req.Identification.Id, func(sn *pb.Subnetwork) {
		sn.Address = req.Update.Address
		sn.PrefixLength = req.Update.PrefixLength
	})

	if err != nil {
		return nil, err
	}

	return subnetwork, nil
}

func (s *SubnetworkService) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Subnetwork]) error {
	subnetworks, errors := s.repository.getAll(stream.Context())

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
