package shared

import (
	"context"
)

// TODO: (This PR) Use a better abstraction for the container model to not expose libcontainer directly.
type ContainerRepository interface {
	Get(id uint32) (*ContainerModel, error)
	GetAll(ctx context.Context) (<-chan *ContainerModel, <-chan error)
	Add(creationModel *ContainerCreationModel) (*ContainerModel, error)
	Delete(id uint32) (*ContainerModel, error)
}
