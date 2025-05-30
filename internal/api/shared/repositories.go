package shared

import (
	"context"
)

type NetworkRepository interface {
	Get(id uint32) (*NetworkModel, error)
	GetAll(ctx context.Context) (<-chan *NetworkModel, <-chan error)
	Add(network *NetworkModel) (*NetworkModel, error)
	Delete(id uint32) error
	Update(id uint32, updateFn func(*NetworkModel)) (*NetworkModel, error)
}

type SubnetworkRepository interface {
	Get(id uint32) (*SubnetworkModel, error)
	GetAll(ctx context.Context) (<-chan *SubnetworkModel, <-chan error)
	GetAllByNetworkId(id uint32, ctx context.Context) (<-chan *SubnetworkModel, <-chan error)
	Add(subnetwork *SubnetworkModel) (*SubnetworkModel, error)
	Delete(id uint32) error
	Update(id uint32, updateFn func(*SubnetworkModel)) (*SubnetworkModel, error)
}
