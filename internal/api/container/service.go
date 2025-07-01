package container

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/BenasB/bx2cloud/internal/api/container/images"
	"github.com/BenasB/bx2cloud/internal/api/id"
	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	runspecs "github.com/opencontainers/runtime-spec/specs-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type service struct {
	pb.UnimplementedContainerServiceServer
	repository           shared.ContainerRepository
	subnetworkRepository shared.SubnetworkRepository
	configurator         configurator
	imagePuller          images.Puller
	ipamRepository       shared.IpamRepository
}

func NewService(
	containerRepository shared.ContainerRepository,
	subnetworkRepository shared.SubnetworkRepository,
	configurator configurator,
	imagePuller images.Puller,
	ipamRepository shared.IpamRepository,
) *service {
	return &service{
		repository:           containerRepository,
		subnetworkRepository: subnetworkRepository,
		configurator:         configurator,
		imagePuller:          imagePuller,
		ipamRepository:       ipamRepository,
	}
}

func (s *service) Get(ctx context.Context, req *pb.ContainerIdentificationRequest) (*pb.Container, error) {
	container, err := s.repository.Get(req.Id)
	if err != nil {
		return nil, err
	}

	return mapModelToDto(container)
}

func (s *service) Delete(ctx context.Context, req *pb.ContainerIdentificationRequest) (*emptypb.Empty, error) {
	container, err := s.repository.Get(req.Id)
	if err != nil {
		return nil, err
	}

	state, err := container.GetState()
	if err != nil {
		log.Printf("Will skip killing the container process, since we can't determine if the container is in a running status: %v", err)
	}

	if err == nil && state.Status == runspecs.StateRunning {
		if err := container.Stop(); err != nil {
			return nil, err
		}
	}

	data := container.GetData()

	subnetwork, err := s.subnetworkRepository.Get(data.SubnetworkId)
	if err != nil {
		return nil, err
	}

	if err := s.configurator.Unconfigure(container, subnetwork); err != nil {
		return nil, err
	}

	if err := s.imagePuller.RemoveRootFs(data.Id); err != nil {
		return nil, err
	}

	if err := s.ipamRepository.Deallocate(subnetwork, data.Ip); err != nil {
		return nil, fmt.Errorf("failed to deallocate an IP for the container: %w", err)
	}

	_, err = s.repository.Delete(data.Id)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *service) Create(ctx context.Context, req *pb.ContainerCreationRequest) (*pb.Container, error) {
	subnetwork, err := s.subnetworkRepository.Get(req.SubnetworkId)
	if err != nil {
		return nil, err
	}

	id := id.NextId("container")

	imgMetadata, err := s.imagePuller.GatherImageMetadata(req.Image)
	if err != nil {
		return nil, err
	}

	rootFsDir, err := s.imagePuller.PrepareRootFs(id, imgMetadata)
	if err != nil {
		return nil, err
	}

	ip, err := s.ipamRepository.Allocate(subnetwork, shared.IPAM_CONTAINER)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate a new IP for the container: %w", err)
	}

	if len(req.Entrypoint) > 0 {
		imgMetadata.Image.Config.Entrypoint = req.Entrypoint
	}

	if len(req.Cmd) > 0 {
		imgMetadata.Image.Config.Cmd = req.Cmd
	}

	if len(req.Env) > 0 {
		imgMetadata.Image.Config.Env = append(imgMetadata.Image.Config.Env, req.Env...)
	}

	spec := imageSpecToRuntimeSpec(id, rootFsDir, &imgMetadata.Image.Config)
	creationModel := &shared.ContainerCreationModel{
		Id:           id,
		Ip:           ip,
		SubnetworkId: subnetwork.Id,
		Image:        req.Image,
		Spec:         spec,
		CreatedAt:    time.Now(),
	}

	container, err := s.repository.Create(creationModel)
	if err != nil {
		return nil, err
	}

	if err := s.configurator.Configure(container, subnetwork); err != nil {
		return nil, err
	}

	if err := container.Exec(); err != nil {
		return nil, err
	}

	return mapModelToDto(container)
}

func (s *service) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Container]) error {
	containers, errors := s.repository.GetAll(stream.Context())

	for {
		select {
		case container, ok := <-containers:
			if !ok {
				select {
				case err := <-errors:
					return err
				default:
					return nil
				}
			}
			dto, err := mapModelToDto(container)
			if err != nil {
				return err
			}
			if err := stream.Send(dto); err != nil {
				return err
			}
		case err, ok := <-errors:
			if ok {
				return err
			}
		}
	}
}

func (s *service) Start(ctx context.Context, req *pb.ContainerIdentificationRequest) (*pb.Container, error) {
	container, err := s.repository.Get(req.Id)
	if err != nil {
		return nil, err
	}

	data := container.GetData()
	state, err := container.GetState()
	if err != nil {
		return nil, err
	}

	if state.Status != runspecs.StateStopped {
		return nil, fmt.Errorf("can't start a container that is not %q", runspecs.StateStopped)
	}

	subnetwork, err := s.subnetworkRepository.Get(data.SubnetworkId)
	if err != nil {
		return nil, err
	}

	if err := s.configurator.Unconfigure(container, subnetwork); err != nil {
		return nil, err
	}

	if _, err := s.repository.Delete(data.Id); err != nil {
		return nil, err
	}

	creationModel := &shared.ContainerCreationModel{
		Id:           data.Id,
		Ip:           data.Ip,
		SubnetworkId: subnetwork.Id,
		Image:        data.Image,
		Spec:         data.Spec,
		CreatedAt:    data.CreatedAt,
	}

	newContainer, err := s.repository.Create(creationModel)
	if err != nil {
		return nil, err
	}

	if err := s.configurator.Configure(newContainer, subnetwork); err != nil {
		return nil, err
	}

	if err := newContainer.Exec(); err != nil {
		return nil, err
	}

	return mapModelToDto(newContainer)
}

func (s *service) Stop(ctx context.Context, req *pb.ContainerIdentificationRequest) (*pb.Container, error) {
	container, err := s.repository.Get(req.Id)
	if err != nil {
		return nil, err
	}

	state, err := container.GetState()
	if err != nil {
		return nil, err
	}

	if state.Status != runspecs.StateRunning {
		return nil, fmt.Errorf("can't stop a container that is not %q", runspecs.StateRunning)
	}

	if err := container.Stop(); err != nil {
		return nil, err
	}

	return mapModelToDto(container)
}

func mapModelToDto(container shared.ContainerModel) (*pb.Container, error) {
	state, err := container.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the container's state: %w", err)
	}

	data := container.GetData()

	address := uint32(data.Ip.IP[0])<<24 | uint32(data.Ip.IP[1])<<16 | uint32(data.Ip.IP[2])<<8 | uint32(data.Ip.IP[3])
	prefixLength, _ := data.Ip.Mask.Size()

	return &pb.Container{
		Id:           data.Id,
		Address:      address,
		PrefixLength: uint32(prefixLength),
		Status:       string(state.Status),
		Image:        data.Image,
		StartedAt:    timestamppb.New(data.StartedAt),
		CreatedAt:    timestamppb.New(data.CreatedAt),
		SubnetworkId: data.SubnetworkId,
	}, nil
}
