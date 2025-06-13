package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/BenasB/bx2cloud/internal/api/id"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	_ "github.com/opencontainers/cgroups/devices"
	"github.com/opencontainers/runc/libcontainer"
	"github.com/opencontainers/runc/libcontainer/specconv"
	"github.com/opencontainers/runtime-spec/specs-go"
)

var _ shared.ContainerRepository = &libcontainerRepository{}

type libcontainerRepository struct {
	root           string
	ipamRepository shared.IpamRepository
}

func NewLibcontainerRepository(ipamRepository shared.IpamRepository) (shared.ContainerRepository, error) {
	root := "/var/run/bx2cloud"
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}

	os.ReadDir(root)
	list, err := os.ReadDir(root)
	if err != nil {
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
		root:           root,
		ipamRepository: ipamRepository,
	}, nil
}

func (r *libcontainerRepository) Get(id uint32) (*shared.ContainerModel, error) {
	return libcontainer.Load(r.root, strconv.FormatInt(int64(id), 10))
}

func (r *libcontainerRepository) GetAll(ctx context.Context) (<-chan *shared.ContainerModel, <-chan error) {
	results := make(chan *shared.ContainerModel, 0)
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

			select {
			case results <- container:
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return results, errChan
}

// Returns a container in a started state
func (r *libcontainerRepository) Add(image string, subnetwork *shared.SubnetworkModel) (*shared.ContainerModel, error) {
	id := id.NextId("container")

	ip, err := r.ipamRepository.Allocate(subnetwork, shared.IPAM_CONTAINER)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate a new IP for the container: %w", err)
	}

	spec := &specs.Spec{
		Version: specs.Version,
		Root: &specs.Root{
			Path:     "/ubuntu-rootfs", // TODO: (This PR) Initialize a rootfs for each container
			Readonly: false,
		},
		Process: &specs.Process{
			Args: []string{"/proc/self/exe", "init"},
			Env:  []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			Cwd:  "/",
		},
		Mounts: []specs.Mount{
			{
				Destination: "/proc",
				Type:        "proc",
				Source:      "proc",
				Options:     []string{"nosuid", "noexec", "nodev"},
			},
			{
				Destination: "/dev/pts",
				Type:        "devpts",
				Source:      "devpts",
				Options:     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620"},
			},
			{
				Destination: "/etc/resolv.conf",
				Type:        "bind",
				Source:      "/etc/resolv.conf",
				Options:     []string{"rbind", "ro"},
			},
		},
		Linux: &specs.Linux{
			Namespaces: []specs.LinuxNamespace{
				{Type: specs.PIDNamespace},
				{Type: specs.IPCNamespace},
				{Type: specs.UTSNamespace},
				{Type: specs.MountNamespace},
				{Type: specs.NetworkNamespace},
			},
		},
		Annotations: map[string]string{
			"image":        image,
			"subnetworkId": strconv.FormatUint(uint64(subnetwork.Id), 10),
			"ip":           ip.String(),
		},
		Hostname: fmt.Sprintf("container-%d", id),
	}

	config, err := specconv.CreateLibcontainerConfig(&specconv.CreateOpts{
		CgroupName:       fmt.Sprintf("bx2cloud-container-%d", id),
		UseSystemdCgroup: false,
		NoPivotRoot:      false,
		NoNewKeyring:     false,
		Spec:             spec,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create the libcontainer config: %w", err)
	}

	container, err := libcontainer.Create(
		r.root,
		strconv.FormatInt(int64(id), 10),
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the container: %w", err)
	}

	initProcess := &libcontainer.Process{
		Args: []string{"sleep", "infinity"},
		Env: []string{
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		},
		Init: true,
	}

	if err := container.Start(initProcess); err != nil {
		return nil, fmt.Errorf("failed to run the container: %w", err)
	}

	return container, nil
}

func (r *libcontainerRepository) Delete(id uint32) (*shared.ContainerModel, error) {
	container, err := r.Get(id)
	if err != nil {
		return nil, err
	}

	if err := container.Destroy(); err != nil {
		return nil, err
	}

	return container, nil
}
