package shared

import (
	"context"

	pb "github.com/BenasB/bx2cloud/internal/api"
)

type NetworkRepository interface {
	Get(id uint32) (*pb.Network, error)
	GetAll(ctx context.Context) (<-chan *pb.Network, <-chan error)
	Add(network *pb.Network) (*pb.Network, error)
	Delete(id uint32) error
	Update(id uint32, updateFn func(*pb.Network)) (*pb.Network, error)
}

type SubnetworkRepository interface {
	Get(id uint32) (*pb.Subnetwork, error)
	GetAll(ctx context.Context) (<-chan *pb.Subnetwork, <-chan error)
	GetAllByNetworkId(id uint32, ctx context.Context) (<-chan *pb.Subnetwork, <-chan error)
	Add(subnetwork *pb.Subnetwork) (*pb.Subnetwork, error)
	Delete(id uint32) error
	Update(id uint32, updateFn func(*pb.Subnetwork)) (*pb.Subnetwork, error)
}
