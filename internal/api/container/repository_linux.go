package container

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BenasB/bx2cloud/internal/api/id"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	_ "github.com/opencontainers/cgroups/devices"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/specconv"
	"github.com/opencontainers/runc/libcontainer/utils"
	runspecs "github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

var _ shared.ContainerRepository = &libcontainerRepository{}

type libcontainerRepository struct {
	root string
}

func NewLibcontainerRepository() (shared.ContainerRepository, error) {
	root := "/var/run/bx2cloud"
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}

	list, err := os.ReadDir(root)
	if err != nil && err != io.EOF {
		return nil, err
	}

	var maxId *uint32
	for _, item := range list {
		if !item.IsDir() {
			continue
		}

		f, err := os.Open(filepath.Join(root, item.Name()))
		if err != nil {
			return nil, err
		}
		defer f.Close()

		_, err = f.Readdirnames(1)
		if err == io.EOF { // Is the container state directory empty
			if err := os.Remove(f.Name()); err != nil {
				return nil, fmt.Errorf("failed to remove an empty container state directory: %w", err)
			}
			continue
		}
		if err != nil {
			return nil, err
		}

		id64, err := strconv.ParseUint(item.Name(), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to convert a container state directory name to an integer id: %w", err)
		}
		id := uint32(id64)
		if maxId == nil || *maxId < id {
			maxId = &id
		}
	}

	if maxId != nil {
		_ = id.Skip("container", *maxId)
	}

	return &libcontainerRepository{
		root: root,
	}, nil
}

func (r *libcontainerRepository) mapToContainerModel(container *libcontainer.Container) (shared.ContainerModel, error) {
	id64, err := strconv.ParseUint(container.ID(), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the container's id: %w", err)
	}

	state, err := container.State()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the container's state: %w", err)
	}

	data := &shared.ContainerModelData{
		Id:        uint32(id64),
		CreatedAt: state.Created,
	}
	var subnetworkId *uint32
	for _, label := range container.Config().Labels {
		if after, found := strings.CutPrefix(label, "image="); found {
			data.Image = after
			continue
		}

		if after, found := strings.CutPrefix(label, "ip="); found {
			ip, ipNet, err := net.ParseCIDR(after)
			if err != nil {
				return nil, fmt.Errorf("failed to parse the container's IP: %w", err)
			}

			data.Ip = &net.IPNet{
				IP:   ip.To4(),
				Mask: ipNet.Mask,
			}
			continue
		}

		if after, found := strings.CutPrefix(label, "subnetworkId="); found {
			id64, err := strconv.ParseUint(after, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse the container's subnetwork id: %w", err)
			}
			id32 := uint32(id64)
			subnetworkId = &id32
			continue
		}

		if after, found := strings.CutPrefix(label, "spec="); found {
			spec := &runspecs.Spec{}
			if err := json.Unmarshal([]byte(after), spec); err != nil {
				return nil, fmt.Errorf("failed to unmarshal OCI specification: %w", err)
			}
			data.Spec = spec
			continue
		}
	}

	if data.Image == "" {
		return nil, fmt.Errorf("failed to locate metadata about the container's image")
	}

	if data.Ip == nil {
		return nil, fmt.Errorf("failed to locate metadata about the container's ip")
	}

	if subnetworkId == nil {
		return nil, fmt.Errorf("failed to locate metadata about the container's subnetworkId")
	}
	data.SubnetworkId = *subnetworkId

	if data.Spec == nil {
		return nil, fmt.Errorf("failed to locate metadata about the container's OCI specification")
	}

	return &wrappedContainer{
		data:      data,
		container: container,
	}, nil
}

func (r *libcontainerRepository) Get(id uint32) (shared.ContainerModel, error) {
	container, err := libcontainer.Load(r.root, strconv.FormatInt(int64(id), 10))

	if err != nil {
		return nil, err
	}

	return r.mapToContainerModel(container)
}

func (r *libcontainerRepository) GetAll(ctx context.Context) (<-chan shared.ContainerModel, <-chan error) {
	results := make(chan shared.ContainerModel, 0)
	errChan := make(chan error, 1)

	go func() {
		defer close(results)
		defer close(errChan)

		list, err := os.ReadDir(r.root)
		if err != nil {
			errChan <- err
			return
		}

		for _, item := range list {
			if !item.IsDir() {
				continue
			}

			container, err := libcontainer.Load(r.root, item.Name())
			if err != nil {
				errChan <- err
				return
			}

			model, err := r.mapToContainerModel(container)
			if err != nil {
				errChan <- err
				return
			}

			select {
			case results <- model:
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return results, errChan
}

func (r *libcontainerRepository) Create(creationModel *shared.ContainerCreationModel) (shared.ContainerModel, error) {
	config, err := specconv.CreateLibcontainerConfig(&specconv.CreateOpts{
		CgroupName:       fmt.Sprintf("bx2cloud-container-%d", creationModel.Id),
		UseSystemdCgroup: false,
		NoPivotRoot:      false,
		NoNewKeyring:     false,
		Spec:             creationModel.Spec,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create the libcontainer config: %w", err)
	}

	serializedSpec, err := json.Marshal(creationModel.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize the OCI specification: %w", err)
	}

	config.Labels = append(config.Labels, fmt.Sprintf("image=%s", creationModel.Image))
	config.Labels = append(config.Labels, fmt.Sprintf("subnetworkId=%d", creationModel.SubnetworkId))
	config.Labels = append(config.Labels, fmt.Sprintf("ip=%s", creationModel.Ip.String()))
	config.Labels = append(config.Labels, fmt.Sprintf("spec=%s", serializedSpec))

	container, err := r.createContainer(creationModel.Id, config, creationModel.Spec.Process)
	if err != nil {
		return nil, err
	}

	return r.mapToContainerModel(container)
}

func (r *libcontainerRepository) createContainer(id uint32, config *configs.Config, init *runspecs.Process) (*libcontainer.Container, error) {
	container, err := libcontainer.Create(
		r.root,
		strconv.FormatInt(int64(id), 10),
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the container: %w", err)
	}

	initProcess := &libcontainer.Process{
		Args: init.Args,
		Env:  init.Env,
		Cwd:  init.Cwd,
		UID:  int(init.User.UID),
		GID:  int(init.User.GID),
		// Not everything is mapped here (yet?)
		Init: true,
	}

	if err := container.Start(initProcess); err != nil {
		return nil, fmt.Errorf("failed to run the container: %w", err)
	}

	return container, nil
}

func (r *libcontainerRepository) Delete(id uint32) (shared.ContainerModel, error) {
	container, err := libcontainer.Load(r.root, strconv.FormatInt(int64(id), 10))
	if err != nil {
		return nil, err
	}

	if err := container.Destroy(); err != nil {
		return nil, err
	}

	return r.mapToContainerModel(container)
}

type signalable interface {
	Signal(os.Signal) error
}

// Wraps the libcontainer.Container implementation to provide a more generic interface used in the contract of the repository
type wrappedContainer struct {
	data      *shared.ContainerModelData
	container *libcontainer.Container
}

func (w *wrappedContainer) GetData() *shared.ContainerModelData {
	return w.data
}

func (w *wrappedContainer) GetState() (*runspecs.State, error) {
	return w.container.OCIState()
}

func (w *wrappedContainer) Exec() error {
	return w.container.Exec()
}

func (w *wrappedContainer) StartInteractive(spec *runspecs.Process) (shared.ContainerInteractiveProcess, error) {
	status, err := w.container.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the container's status: %w", err)
	}

	if status != libcontainer.Running {
		return nil, fmt.Errorf("the container is not running")
	}

	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create a socket pair for console fd retrieval: %w", err)
	}
	parentConsoleSocket := os.NewFile(uintptr(fds[1]), "parent-console-socket")
	childConsoleSocket := os.NewFile(uintptr(fds[0]), "child-console-socket")
	defer parentConsoleSocket.Close()
	defer childConsoleSocket.Close()

	process := &libcontainer.Process{
		Args:          spec.Args,
		Env:           spec.Env,
		ConsoleSocket: childConsoleSocket,
		ConsoleWidth:  uint16(spec.ConsoleSize.Width),
		ConsoleHeight: uint16(spec.ConsoleSize.Height),
		Init:          false,
	}

	if err := w.container.Start(process); err != nil {
		return nil, fmt.Errorf("failed to start the container process: %w", err)
	}

	ptyMaster, err := utils.RecvFile(parentConsoleSocket)
	if err != nil {
		return nil, fmt.Errorf("failed to receive console master fd: %w", err)
	}

	return &wrappedProcess{
		pty:     ptyMaster,
		process: process,
	}, nil
}

func (w *wrappedContainer) Stop() error {
	status, err := w.container.Status()
	if err != nil {
		return fmt.Errorf("failed to retrieve the container's status: %w", err)
	}

	if status != libcontainer.Running {
		return fmt.Errorf("can't stop a container that is not running")
	}

	return stopSignalable(w.container)
}

// Wraps the libcontainer.Process implementation to provide a more generic interface used in the contract of the repository
type wrappedProcess struct {
	pty     *os.File
	process *libcontainer.Process
}

func (w *wrappedProcess) GetPty() *os.File {
	return w.pty
}

func (w *wrappedProcess) Wait() (int, error) {
	state, err := w.process.Wait()
	if err != nil {
		return 0, err
	}

	return state.ExitCode(), nil
}

func (w *wrappedProcess) Stop() error {
	if err := w.process.Signal(unix.Signal(0)); err == nil {
		return fmt.Errorf("the process is not running")
	}

	return stopSignalable(w.process)
}

func stopSignalable(s signalable) error {
	// TODO: Send SIGTERM first to try to gracefully shut down the process
	if err := s.Signal(unix.SIGKILL); err != nil {
		return fmt.Errorf("failed to send a kill signal to the container process: %w", err)
	}

	for range 100 {
		time.Sleep(100 * time.Millisecond)
		if err := s.Signal(unix.Signal(0)); err != nil {
			return nil
		}
	}

	return fmt.Errorf("failed to kill the container process")
}
