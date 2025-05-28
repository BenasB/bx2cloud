package network

import (
	"context"
	"fmt"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/id"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type networkRepository interface {
	get(id uint32) (*pb.Network, error)
	getAll(ctx context.Context) (<-chan *pb.Network, <-chan error)
	add(network *pb.Network) (*pb.Network, error)
	delete(id uint32) error
	update(id uint32, updateFn func(*pb.Network)) (*pb.Network, error)
}

var _ networkRepository = &memoryNetworkRepository{}

// Caution: not thread safe
type memoryNetworkRepository struct {
	networks []*pb.Network
}

func NewMemoryNetworkRepository(networks []*pb.Network) networkRepository {
	sns := make([]*pb.Network, len(networks))
	for i, network := range networks {
		sns[i] = proto.Clone(network).(*pb.Network)
	}

	return &memoryNetworkRepository{
		networks: sns,
	}
}

func (r *memoryNetworkRepository) get(id uint32) (*pb.Network, error) {
	for _, network := range r.networks {
		if network.Id == id {
			return network, nil
		}
	}

	return nil, fmt.Errorf("could not find network")
}

func (r *memoryNetworkRepository) getAll(ctx context.Context) (<-chan *pb.Network, <-chan error) {
	results := make(chan *pb.Network, 0)
	errChan := make(chan error, 1)

	go func() {
		defer close(results)
		defer close(errChan)

		for _, network := range r.networks {
			select {
			case results <- network:
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return results, errChan
}

func (r *memoryNetworkRepository) add(network *pb.Network) (*pb.Network, error) {
	//newNetwork := *network
	newNetwork := proto.Clone(network).(*pb.Network)
	newNetwork.Id = id.NextId("network")
	newNetwork.CreatedAt = timestamppb.New(time.Now())
	r.networks = append(r.networks, newNetwork)
	return newNetwork, nil
}

func (r *memoryNetworkRepository) delete(id uint32) error {
	for i, network := range r.networks {
		if network.Id == id {
			r.networks = append(r.networks[:i], r.networks[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("could not find network")
}

func (r *memoryNetworkRepository) update(id uint32, updateFn func(*pb.Network)) (*pb.Network, error) {
	for _, network := range r.networks {
		if network.Id == id {
			updateFn(network)
			return network, nil
		}
	}

	return nil, fmt.Errorf("could not find network")
}
