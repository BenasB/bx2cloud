package subnetwork

import (
	"context"
	"fmt"

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
	ipamRepository    shared.IpamRepository
}

func NewService(
	subnetworkRepository shared.SubnetworkRepository,
	networkRepository shared.NetworkRepository,
	configurator configurator,
	ipamRepository shared.IpamRepository,
) *service {
	return &service{
		repository:        subnetworkRepository,
		networkRepository: networkRepository,
		configurator:      configurator,
		ipamRepository:    ipamRepository,
	}
}

func (s *service) Get(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*pb.Subnetwork, error) {
	return s.repository.Get(req.Id)
}

func (s *service) Delete(ctx context.Context, req *pb.SubnetworkIdentificationRequest) (*emptypb.Empty, error) {
	subnetwork, err := s.repository.Get(req.Id)
	if err != nil {
		return nil, err
	}

	if alloc, found := s.ipamRepository.HasAllocations(subnetwork); found {
		switch alloc {
		case shared.IPAM_CONTAINER:
			return nil, fmt.Errorf("the subnetwork still has an IP allocated for a container")
		default:
			return nil, fmt.Errorf("the subnetwork still has an IP allocated for a resource")
		}
	}

	_, err = s.repository.Delete(subnetwork.Id)
	if err != nil {
		return nil, err
	}

	if err := s.configurator.Unconfigure(subnetwork); err != nil {
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

	subnetworks, errors := s.repository.GetAllByNetworkId(req.NetworkId, ctx)

	err := func() error {
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
				} else {
					minPrefix := min(newSubnetwork.PrefixLength, subnetwork.PrefixLength)
					a := newSubnetwork.Address & minPrefix
					b := subnetwork.Address & minPrefix
					if a == b {
						return fmt.Errorf("new subnetwork would overlap with subnetwork %d", subnetwork.Id)
					}
				}
			case err, ok := <-errors:
				if ok {
					return err
				}
			}
		}
	}()

	if err != nil {
		return nil, err
	}

	returnedSubnetwork, err := s.repository.Add(newSubnetwork)
	if err != nil {
		return nil, err
	}

	if err := s.configurator.Configure(returnedSubnetwork); err != nil {
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

	if err := s.configurator.Configure(subnetwork); err != nil {
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
