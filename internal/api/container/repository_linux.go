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
func (r *libcontainerRepository) Add(creationModel *shared.ContainerCreationModel) (*shared.ContainerModel, error) {
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

	config.Labels = append(config.Labels, fmt.Sprintf("image=%s", creationModel.Image))
	config.Labels = append(config.Labels, fmt.Sprintf("subnetworkId=%d", creationModel.SubnetworkId))
	config.Labels = append(config.Labels, fmt.Sprintf("ip=%s", creationModel.Ip.String()))

	container, err := libcontainer.Create(
		r.root,
		strconv.FormatInt(int64(creationModel.Id), 10),
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the container: %w", err)
	}

	initProcess := &libcontainer.Process{
		Args: creationModel.Spec.Process.Args,
		Env:  creationModel.Spec.Process.Env,
		Cwd:  creationModel.Spec.Process.Cwd,
		UID:  int(creationModel.Spec.Process.User.UID),
		GID:  int(creationModel.Spec.Process.User.GID),
		// Not everything is mapped here (yet?)
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
