package shared

import (
	"context"
)

// TODO: (This PR) Use a better abstraction for the container model to not expose libcontainer directly.
type ContainerRepository interface {
	Get(id uint32) (*ContainerModel, error)
	GetAll(ctx context.Context) (<-chan *ContainerModel, <-chan error)
	Add(id uint32, image string, rootFsDir string, subnetwork *SubnetworkModel) (*ContainerModel, error)
	Delete(id uint32) (*ContainerModel, error)
}
