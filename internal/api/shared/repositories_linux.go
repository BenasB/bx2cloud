package shared

import (
	"context"

	runspecs "github.com/opencontainers/runtime-spec/specs-go"
)

// TODO: (This PR) Use a better abstraction for the container model to not expose libcontainer directly.
type ContainerRepository interface {
	Get(id uint32) (*ContainerModel, error)
	GetAll(ctx context.Context) (<-chan *ContainerModel, <-chan error)
	Add(id uint32, spec *runspecs.Spec) (*ContainerModel, error)
	Delete(id uint32) (*ContainerModel, error)
}
