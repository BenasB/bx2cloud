package container

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/utils"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	pb.UnimplementedContainerServiceServer
	repository shared.ContainerRepository
}

func NewService(repository shared.ContainerRepository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Get(ctx context.Context, req *pb.ContainerIdentificationRequest) (*pb.Container, error) {
	container, err := s.repository.Get(req.Id)
	if err != nil {
		return nil, err
	}

	return mapModelToDto(container)
}

func (s *Service) Delete(ctx context.Context, req *pb.ContainerIdentificationRequest) (*emptypb.Empty, error) {
	_, err := s.repository.Delete(req.Id)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *Service) Create(ctx context.Context, req *pb.ContainerCreationRequest) (*pb.Container, error) {
	container, err := s.repository.Add("ubuntu:24.04")
	if err != nil {
		return nil, err
	}

	return mapModelToDto(container)
}

func (s *Service) List(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Container]) error {
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

func (s *Service) Exec(stream pb.ContainerService_ExecServer) error {
	first, err := stream.Recv() // Expect StartExec first
	if err != nil {
		return err
	}
	request := first.GetInitialization()
	if request == nil {
		return fmt.Errorf("First message in the stream is expected to initialize the command")
	}

	container, err := s.repository.Get(request.Identification.Id)
	if err != nil {
		return err
	}

	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	parentSocket := fds[1]
	childSocket := fds[0]
	if err != nil {
		return fmt.Errorf("failed to create socket pair: %w", err)
	}

	parentConsoleSocket := os.NewFile(uintptr(parentSocket), "parent-console-socket")
	childConsoleSocket := os.NewFile(uintptr(childSocket), "child-console-socket")
	defer parentConsoleSocket.Close()
	defer childConsoleSocket.Close()

	process := &libcontainer.Process{
		Args: []string{"/bin/sh"},
		Env: []string{
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		ConsoleSocket: childConsoleSocket,
		ConsoleWidth:  uint16(request.ConsoleWidth),
		ConsoleHeight: uint16(request.ConsoleHeight),
		Init:          false,
	}

	if err := container.Run(process); err != nil {
		return err
	}

	ptyMaster, err := utils.RecvFile(parentConsoleSocket)
	if err != nil {
		return fmt.Errorf("failed to receive pty master fd: %w", err)
	}
	parentConsoleSocket.Close()
	childConsoleSocket.Close()
	defer ptyMaster.Close()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptyMaster.Read(buf)
			if n > 0 {
				log.Printf("Bytes sent: %d\n", n)
				stream.Send(&pb.ContainerExecResponse{
					Output: &pb.ContainerExecResponse_Stdout{Stdout: buf[:n]},
				})
			}
			if err != nil {
				log.Print(err)
				break
			}
		}
	}()

	for {
		req, err := stream.Recv()
		if err != nil {
			break
		}

		switch p := req.Input.(type) {
		case *pb.ContainerExecRequest_Stdin:
			log.Printf("Bytes received: %d\n", len(p.Stdin))
			if _, err := ptyMaster.Write(p.Stdin); err != nil {
				return err
			}
		}
	}

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
	for _, label := range config.Labels {
		after, found := strings.CutPrefix(label, "image=")
		if found {
			image = after
			break
		}
	}

	if image == "" {
		return nil, fmt.Errorf("failed to locate metadata about the container's image")
	}

	return &pb.Container{
		Id:           uint32(id),
		Address:      0, // TODO
		PrefixLength: 0, // TODO
		Status:       int32(status),
		Image:        image,
		CreatedAt:    timestamppb.New(state.Created),
	}, nil
}
