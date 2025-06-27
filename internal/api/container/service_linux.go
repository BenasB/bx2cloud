package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"

	"github.com/BenasB/bx2cloud/internal/api/container/images"
	"github.com/BenasB/bx2cloud/internal/api/id"
	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	runspecs "github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
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

	spec := imageSpecToRuntimeSpec(id, rootFsDir, &imgMetadata.Image.Config)
	creationModel := &shared.ContainerCreationModel{
		Id:           id,
		Ip:           ip,
		SubnetworkId: subnetwork.Id,
		Image:        req.Image,
		Spec:         spec,
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

func (s *service) Exec(stream pb.ContainerService_ExecServer) error {
	first, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to read the stream to retrieve the first stream message: %w", err)
	}
	init := first.GetInitialization()
	if init == nil {
		return fmt.Errorf("first message in the stream is expected to be an initialization message")
	}

	container, err := s.repository.Get(init.Identification.Id)
	if err != nil {
		return fmt.Errorf("failed to retrieve the container for command execution: %w", err)
	}

	term := "xterm"
	if init.Terminal != nil {
		term = *init.Terminal
	}

	spec := &runspecs.Process{
		Args: []string{
			"/bin/sh",
			"-c",
			"[ -x /bin/bash ] && exec /bin/bash || exec /bin/sh",
		},
		Env: []string{
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			fmt.Sprintf("TERM=%s", term),
		},
		ConsoleSize: &runspecs.Box{
			Width:  uint(init.ConsoleWidth),
			Height: uint(init.ConsoleHeight),
		},
	}

	process, err := container.StartInteractive(spec)
	if err != nil {
		return fmt.Errorf("failed to start an interactive console session: %w", err)
	}
	pty := process.GetPty()
	defer pty.Close()

	log.Printf("Established an interactive console session with container id %d", init.Identification.Id)

	results := make(chan error, 2)
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := pty.Read(buf)
			if errors.Is(err, unix.EIO) {
				// pty child was closed, which is considered a successfull result
				results <- nil
				return
			}
			if err != nil {
				results <- fmt.Errorf("failed to read master console: %w", err)
				return
			}
			if n > 0 {
				err = stream.Send(&pb.ContainerExecResponse{
					Output: &pb.ContainerExecResponse_Stdout{Stdout: buf[:n]},
				})
				if err != nil {
					results <- fmt.Errorf("failed to send bytes from the master console to the client: %w", err)
					return
				}
			}
		}
	}()

	go func() {
		for {
			req, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				results <- fmt.Errorf("client disconnected: %w", err)
				return
			}
			if err != nil {
				results <- fmt.Errorf("failed to read from the client stream: %w", err)
				return
			}

			if p := req.GetStdin(); p != nil {
				if _, err := pty.Write(p); err != nil {
					results <- fmt.Errorf("failed to write to the master console: %w", err)
					return
				}
			}
		}
	}()

	err = <-results

	if err != nil {
		if stopErr := process.Stop(); stopErr != nil {
			return fmt.Errorf("failed to kill the exec process: %w, when the original error was: %w", stopErr, err)
		}
	}

	exitCode, err := process.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return fmt.Errorf("failed to retrieve the state of the process: %w", err)
		}
	}

	err = stream.Send(&pb.ContainerExecResponse{
		Output: &pb.ContainerExecResponse_ExitCode{
			ExitCode: int32(exitCode),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to send the exit code: %w", err)
	}

	log.Printf("Successfully finished an interactive console session with container id %d", init.Identification.Id)

	return nil
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
		CreatedAt:    timestamppb.New(data.CreatedAt),
	}, nil
}
