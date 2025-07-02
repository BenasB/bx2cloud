package subnetwork

import (
	"context"
	"fmt"
	"time"

	"github.com/BenasB/bx2cloud/internal/api/id"
	"github.com/BenasB/bx2cloud/internal/api/interfaces"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ interfaces.SubnetworkRepository = &memoryRepository{}

// Caution: not thread safe
type memoryRepository struct {
	subnetworks []*interfaces.SubnetworkModel
}

func NewMemoryRepository(subnetworks []*interfaces.SubnetworkModel) interfaces.SubnetworkRepository {
	sns := make([]*interfaces.SubnetworkModel, len(subnetworks))
	for i, subnetwork := range subnetworks {
		sns[i] = proto.Clone(subnetwork).(*interfaces.SubnetworkModel)
	}

	return &memoryRepository{
		subnetworks: sns,
	}
}

func (r *memoryRepository) Get(id uint32) (*interfaces.SubnetworkModel, error) {
	for _, subnetwork := range r.subnetworks {
		if subnetwork.Id == id {
			return subnetwork, nil
		}
	}

	return nil, fmt.Errorf("could not find subnetwork with id %d", id)
}

func (r *memoryRepository) GetAll(ctx context.Context) (<-chan *interfaces.SubnetworkModel, <-chan error) {
	results := make(chan *interfaces.SubnetworkModel, 0)
	errChan := make(chan error, 1)

	go func() {
		defer close(results)
		defer close(errChan)

		for _, subnetwork := range r.subnetworks {
			select {
			case results <- subnetwork:
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return results, errChan
}

func (r *memoryRepository) GetAllByNetworkId(id uint32, ctx context.Context) (<-chan *interfaces.SubnetworkModel, <-chan error) {
	results := make(chan *interfaces.SubnetworkModel, 0)
	errChan := make(chan error, 1)

	go func() {
		defer close(results)
		defer close(errChan)

		for _, subnetwork := range r.subnetworks {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			if subnetwork.NetworkId != id {
				continue
			}

			select {
			case results <- subnetwork:
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return results, errChan
}

func (r *memoryRepository) Add(subnetwork *interfaces.SubnetworkModel) (*interfaces.SubnetworkModel, error) {
	//newSubnetwork := *subnetwork
	newSubnetwork := proto.Clone(subnetwork).(*interfaces.SubnetworkModel)
	newSubnetwork.Id = id.NextId("subnetwork")
	newSubnetwork.CreatedAt = timestamppb.New(time.Now())
	r.subnetworks = append(r.subnetworks, newSubnetwork)
	return newSubnetwork, nil
}

func (r *memoryRepository) Delete(id uint32) (*interfaces.SubnetworkModel, error) {
	for i, subnetwork := range r.subnetworks {
		if subnetwork.Id == id {
			r.subnetworks = append(r.subnetworks[:i], r.subnetworks[i+1:]...)
			return subnetwork, nil
		}
	}

	return nil, fmt.Errorf("could not find subnetwork with id %d", id)
}

func (r *memoryRepository) Update(id uint32, updateFn func(*interfaces.SubnetworkModel)) (*interfaces.SubnetworkModel, error) {
	for _, subnetwork := range r.subnetworks {
		if subnetwork.Id == id {
			updateFn(subnetwork)
			return subnetwork, nil
		}
	}

	return nil, fmt.Errorf("could not find subnetwork with id %d", id)
}
