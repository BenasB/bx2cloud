package subnetwork

import (
	"context"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type service struct {
	pb.UnimplementedSubnetworkServiceServer
	repository        shared.SubnetworkRepository
	networkRepository shared.NetworkRepository
	configurator      configurator
}

func NewService(subnetworkRepository shared.SubnetworkRepository, networkRepository shared.NetworkRepository, configurator configurator) *service {
	return &service{
		repository:        subnetworkRepository,
		networkRepository: networkRepository,
		configurator:      configurator,
	}
}

func (s *service) Get(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*pb.Subnetwork, error) {
	return s.repository.Get(req.Id)
}

func (s *service) Delete(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*emptypb.Empty, error) {
	subnetwork, err := s.repository.Delete(req.Id)
	if err != nil {
		return nil, err
	}

	// TODO: Handle things that are connected to this subnetwork

	if err := s.configurator.unconfigure(subnetwork); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *service) Create(ctx context.Context, req *pb.SubnetworkCreationRequest) (*pb.Subnetwork, error) {
	if _, err := s.networkRepository.Get(req.NetworkId); err != nil {
		return nil, err
	}

	newSubnetwork := &shared.SubnetworkModel{
		NetworkId:    req.NetworkId,
		Address:      req.Address, // TODO: #1 AND address with network mask to make sure this stores the network IP + unit test
		PrefixLength: req.PrefixLength,
	}

	returnedSubnetwork, err := s.repository.Add(newSubnetwork)
	if err != nil {
		return nil, err
	}

	if err := s.configurator.configure(returnedSubnetwork); err != nil {
		return nil, err
	}

	return returnedSubnetwork, nil
}

func (s *service) Update(ctx context.Context, req *pb.SubnetworkUpdateRequest) (*pb.Subnetwork, error) {
	subnetwork, err := s.repository.Update(req.Identification.Id, func(sn *shared.SubnetworkModel) {
		sn.Address = req.Update.Address
		sn.PrefixLength = req.Update.PrefixLength
	})

	if err != nil {
		return nil, err
	}

	if err := s.configurator.configure(subnetwork); err != nil {
		return nil, err
	}

	return subnetwork, nil
}

func (s *service) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Subnetwork]) error {
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
		case err, ok := <-errors:
			if ok {
				return err
			}
		}
	}
}
