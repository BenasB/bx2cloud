package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/BenasB/bx2cloud/internal/api/container/images"
	"github.com/BenasB/bx2cloud/internal/api/id"
	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/utils"
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

	status, err := container.Status()
	if err != nil {
		log.Printf("Will skip killing the container process, since we can't determine if the container is in a running status: %v", err)
	}

	if err == nil && status == libcontainer.Running {
		// TODO: Send SIGTERM first to try to gracefully shut down the process
		if err := container.Signal(unix.SIGKILL); err != nil {
			return nil, fmt.Errorf("failed to send a kill signal to the container process: %w", err)
		}
		processIsRunning := true
		for range 100 {
			time.Sleep(100 * time.Millisecond)
			if err := container.Signal(unix.Signal(0)); err != nil {
				processIsRunning = false
				break // Process is no longer running
			}
		}

		if processIsRunning {
			return nil, fmt.Errorf("failed to kill the container process: %w", err)
		}
	}

	var subnetworkId *uint32
	var containerIpNet *net.IPNet
	for _, label := range container.Config().Labels {
		if after, found := strings.CutPrefix(label, "subnetworkId="); found {
			id64, err := strconv.ParseUint(after, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse the container's subnetwork id: %w", err)
			}
			id32 := uint32(id64)
			subnetworkId = &id32
			continue
		}

		if after, found := strings.CutPrefix(label, "ip="); found {
			ip, ipNet, err := net.ParseCIDR(after)
			if err != nil {
				return nil, fmt.Errorf("failed to parse the container's IP: %w", err)
			}

			containerIpNet = &net.IPNet{
				IP:   ip.To4(),
				Mask: ipNet.Mask,
			}
			continue
		}
	}

	if subnetworkId == nil {
		return nil, fmt.Errorf("failed to retrieve the container's subnetwork id")
	}

	if containerIpNet == nil {
		return nil, fmt.Errorf("failed to retrieve the container's IP")
	}

	subnetwork, err := s.subnetworkRepository.Get(*subnetworkId)
	if err != nil {
		return nil, err
	}

	if err := s.configurator.Unconfigure(container, subnetwork); err != nil {
		return nil, err
	}

	if err := s.imagePuller.RemoveRootFs(req.Id); err != nil {
		return nil, err
	}

	if err := s.ipamRepository.Deallocate(subnetwork, containerIpNet); err != nil {
		return nil, fmt.Errorf("failed to deallocate an IP for the container: %w", err)
	}

	_, err = s.repository.Delete(req.Id)
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

	container, err := s.repository.Add(creationModel)
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

	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("failed to create a socket pair for console fd retrieval: %w", err)
	}
	parentConsoleSocket := os.NewFile(uintptr(fds[1]), "parent-console-socket")
	childConsoleSocket := os.NewFile(uintptr(fds[0]), "child-console-socket")
	defer parentConsoleSocket.Close()
	defer childConsoleSocket.Close()

	term := "xterm"
	if init.Terminal != nil {
		term = *init.Terminal
	}

	process := &libcontainer.Process{
		Args: []string{
			"/bin/sh",
			"-c",
			"[ -x /bin/bash ] && exec /bin/bash || exec /bin/sh",
		},
		Env: []string{
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			fmt.Sprintf("TERM=%s", term),
		},
		ConsoleSocket: childConsoleSocket,
		ConsoleWidth:  uint16(init.ConsoleWidth),
		ConsoleHeight: uint16(init.ConsoleHeight),
		Init:          false,
	}

	if err := container.Start(process); err != nil {
		return fmt.Errorf("failed to start the container process: %w", err)
	}

	ptyMaster, err := utils.RecvFile(parentConsoleSocket)
	if err != nil {
		return fmt.Errorf("failed to receive console master fd: %w", err)
	}
	parentConsoleSocket.Close()
	childConsoleSocket.Close()
	defer ptyMaster.Close()

	log.Printf("Established an interactive console session with container id %d", init.Identification.Id)

	results := make(chan error, 2)
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := ptyMaster.Read(buf)
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
				if _, err := ptyMaster.Write(p); err != nil {
					results <- fmt.Errorf("failed to write to the master console: %w", err)
					return
				}
			}
		}
	}()

	err = <-results

	if err != nil {
		if signalErr := process.Signal(unix.SIGKILL); signalErr != nil {
			return fmt.Errorf("failed to kill the exec process: %w, when the original error was: %w", signalErr, err)
		}
	}

	state, err := process.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return fmt.Errorf("failed to retrieve the state of the process: %w", err)
		}
	}

	err = stream.Send(&pb.ContainerExecResponse{
		Output: &pb.ContainerExecResponse_ExitCode{
			ExitCode: int32(state.ExitCode()),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to send the exit code: %w", err)
	}

	log.Printf("Successfully finished an interactive console session with container id %d", init.Identification.Id)

	return nil
}

func mapModelToDto(container *shared.ContainerModel) (*pb.Container, error) {
	state, err := container.State()
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(state.ID, 10, 32)
	if err != nil {
		return nil, err
	}

	status, err := container.Status()
	if err != nil {
		return nil, err
	}

	config := container.Config()

	var image string
	var containerIpNet *net.IPNet
	for _, label := range config.Labels {
		if after, found := strings.CutPrefix(label, "image="); found {
			image = after
			continue
		}

		if after, found := strings.CutPrefix(label, "ip="); found {
			ip, ipNet, err := net.ParseCIDR(after)
			if err != nil {
				return nil, fmt.Errorf("failed to parse the container's IP: %w", err)
			}

			containerIpNet = &net.IPNet{
				IP:   ip.To4(),
				Mask: ipNet.Mask,
			}
			continue
		}
	}

	if image == "" {
		return nil, fmt.Errorf("failed to locate metadata about the container's image")
	}

	if containerIpNet == nil {
		return nil, fmt.Errorf("failed to locate metadata about the container's ip")
	}

	address := uint32(containerIpNet.IP[0])<<24 | uint32(containerIpNet.IP[1])<<16 | uint32(containerIpNet.IP[2])<<8 | uint32(containerIpNet.IP[3])
	prefixLength, _ := containerIpNet.Mask.Size()

	return &pb.Container{
		Id:           uint32(id),
		Address:      address,
		PrefixLength: uint32(prefixLength),
		Status:       int32(status),
		Image:        image,
		CreatedAt:    timestamppb.New(state.Created),
	}, nil
}
