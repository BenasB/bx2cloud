package subnetwork

import (
	"context"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Service struct {
	pb.UnimplementedSubnetworkServiceServer
	repository        shared.SubnetworkRepository
	networkRepository shared.NetworkRepository
	configurator      configurator
}

func NewService(subnetworkRepository shared.SubnetworkRepository, networkRepository shared.NetworkRepository, configurator configurator) *Service {
	return &Service{
		repository:        subnetworkRepository,
		networkRepository: networkRepository,
		configurator:      configurator,
	}
}

func (s *Service) Get(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*pb.Subnetwork, error) {
	return s.repository.Get(req.Id)
}

func (s *Service) Delete(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*emptypb.Empty, error) {
	subnetwork, err := s.repository.Delete(req.Id)
	if err != nil {
		return nil, err
	}

	network, err := s.networkRepository.Get(subnetwork.NetworkId)
	if err != nil {
		return nil, err
	}

	// TODO: Handle things that are connected to this subnetwork

	if err := s.configurator.unconfigure(subnetwork, network); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) Create(ctx context.Context, req *pb.SubnetworkCreationRequest) (*pb.Subnetwork, error) {
	network, err := s.networkRepository.Get(req.NetworkId)
	if err != nil {
		return nil, err
	}

	newSubnetwork := &shared.SubnetworkModel{
		NetworkId:    req.NetworkId,
		Address:      req.Address, // TODO: AND address with network mask to make sure this stores the network IP + unit test
		PrefixLength: req.PrefixLength,
	}

	returnedSubnetwork, err := s.repository.Add(newSubnetwork)
	if err != nil {
		return nil, err
	}

	if err := s.configurator.configure(returnedSubnetwork, network); err != nil {
		return nil, err
	}

	return returnedSubnetwork, nil
}

func (s *Service) Update(ctx context.Context, req *pb.SubnetworkUpdateRequest) (*pb.Subnetwork, error) {
	subnetwork, err := s.repository.Update(req.Identification.Id, func(sn *shared.SubnetworkModel) {
		sn.Address = req.Update.Address
		sn.PrefixLength = req.Update.PrefixLength
	})

	if err != nil {
		return nil, err
	}

	network, err := s.networkRepository.Get(subnetwork.NetworkId)
	if err != nil {
		return nil, err
	}

	if err := s.configurator.configure(subnetwork, network); err != nil {
		return nil, err
	}

	return subnetwork, nil
}

func (s *Service) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Subnetwork]) error {
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
