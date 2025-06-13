package shared

import (
	"context"
)

type ContainerRepository interface {
	Get(id uint32) (*ContainerModel, error)
	GetAll(ctx context.Context) (<-chan *ContainerModel, <-chan error)
	Add(image string, subnetwork *SubnetworkModel) (*ContainerModel, error)
	Delete(id uint32) (*ContainerModel, error)
}
