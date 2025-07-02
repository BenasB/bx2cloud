package interfaces

import (
	"context"
	"net"
)

type NetworkRepository interface {
	Get(id uint32) (*NetworkModel, error)
	// TODO: Maybe Reader/Writer would work better here than two manually handled channels?
	GetAll(ctx context.Context) (<-chan *NetworkModel, <-chan error)
	Add(network *NetworkModel) (*NetworkModel, error)
	Delete(id uint32) (*NetworkModel, error)
	Update(id uint32, updateFn func(*NetworkModel)) (*NetworkModel, error)
}

type SubnetworkRepository interface {
	Get(id uint32) (*SubnetworkModel, error)
	GetAll(ctx context.Context) (<-chan *SubnetworkModel, <-chan error)
	GetAllByNetworkId(id uint32, ctx context.Context) (<-chan *SubnetworkModel, <-chan error)
	Add(subnetwork *SubnetworkModel) (*SubnetworkModel, error)
	Delete(id uint32) (*SubnetworkModel, error)
	Update(id uint32, updateFn func(*SubnetworkModel)) (*SubnetworkModel, error)
}

type IpamRepository interface {
	GetSubnetworkGateway(subnetwork *SubnetworkModel) *net.IPNet
	Allocate(subnetwork *SubnetworkModel, resourceType IpamType) (*net.IPNet, error)
	Deallocate(subnetwork *SubnetworkModel, ip *net.IPNet) error
	// Returns the first allocation found
	HasAllocations(subnetwork *SubnetworkModel) (IpamType, bool)
}

type ContainerRepository interface {
	Get(id uint32) (ContainerModel, error)
	GetAll(ctx context.Context) (<-chan ContainerModel, <-chan error)
	// Returns a container in a 'created' state
	Create(creationModel *ContainerCreationModel) (ContainerModel, error)
	Delete(id uint32) (ContainerModel, error)
}
