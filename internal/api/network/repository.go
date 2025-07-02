package network

import (
	"context"
	"fmt"
	"time"

	"github.com/BenasB/bx2cloud/internal/api/id"
	"github.com/BenasB/bx2cloud/internal/api/interfaces"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ interfaces.NetworkRepository = &memoryRepository{}

// Caution: not thread safe
type memoryRepository struct {
	networks []*interfaces.NetworkModel
}

func NewMemoryRepository(networks []*interfaces.NetworkModel) interfaces.NetworkRepository {
	sns := make([]*interfaces.NetworkModel, len(networks))
	for i, network := range networks {
		sns[i] = proto.Clone(network).(*interfaces.NetworkModel)
	}

	return &memoryRepository{
		networks: sns,
	}
}

func (r *memoryRepository) Get(id uint32) (*interfaces.NetworkModel, error) {
	for _, network := range r.networks {
		if network.Id == id {
			return network, nil
		}
	}

	return nil, fmt.Errorf("could not find network with id %d", id)
}

func (r *memoryRepository) GetAll(ctx context.Context) (<-chan *interfaces.NetworkModel, <-chan error) {
	results := make(chan *interfaces.NetworkModel, 0)
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

func (r *memoryRepository) Add(network *interfaces.NetworkModel) (*interfaces.NetworkModel, error) {
	//newNetwork := *network
	newNetwork := proto.Clone(network).(*interfaces.NetworkModel)
	newNetwork.Id = id.NextId("network")
	newNetwork.CreatedAt = timestamppb.New(time.Now())
	r.networks = append(r.networks, newNetwork)
	return newNetwork, nil
}

func (r *memoryRepository) Delete(id uint32) (*interfaces.NetworkModel, error) {
	for i, network := range r.networks {
		if network.Id == id {
			r.networks = append(r.networks[:i], r.networks[i+1:]...)
			return network, nil
		}
	}

	return nil, fmt.Errorf("could not find network with id %d", id)
}

func (r *memoryRepository) Update(id uint32, updateFn func(*interfaces.NetworkModel)) (*interfaces.NetworkModel, error) {
	for _, network := range r.networks {
		if network.Id == id {
			updateFn(network)
			return network, nil
		}
	}

	return nil, fmt.Errorf("could not find network with id %d", id)
}
