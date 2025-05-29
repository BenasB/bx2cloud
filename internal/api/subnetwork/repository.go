package subnetwork

import (
	"context"
	"fmt"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/id"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ shared.SubnetworkRepository = &memorySubnetworkRepository{}

// Caution: not thread safe
type memorySubnetworkRepository struct {
	subnetworks []*pb.Subnetwork
}

func NewMemorySubnetworkRepository(subnetworks []*pb.Subnetwork) shared.SubnetworkRepository {
	sns := make([]*pb.Subnetwork, len(subnetworks))
	for i, subnetwork := range subnetworks {
		sns[i] = proto.Clone(subnetwork).(*pb.Subnetwork)
	}

	return &memorySubnetworkRepository{
		subnetworks: sns,
	}
}

func (r *memorySubnetworkRepository) Get(id uint32) (*pb.Subnetwork, error) {
	for _, subnetwork := range r.subnetworks {
		if subnetwork.Id == id {
			return subnetwork, nil
		}
	}

	return nil, fmt.Errorf("could not find subnetwork with id %d", id)
}

func (r *memorySubnetworkRepository) GetAll(ctx context.Context) (<-chan *pb.Subnetwork, <-chan error) {
	results := make(chan *pb.Subnetwork, 0)
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

func (r *memorySubnetworkRepository) GetAllByNetworkId(id uint32, ctx context.Context) (<-chan *pb.Subnetwork, <-chan error) {
	results := make(chan *pb.Subnetwork, 0)
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

func (r *memorySubnetworkRepository) Add(subnetwork *pb.Subnetwork) (*pb.Subnetwork, error) {
	//newSubnetwork := *subnetwork
	newSubnetwork := proto.Clone(subnetwork).(*pb.Subnetwork)
	newSubnetwork.Id = id.NextId("subnetwork")
	newSubnetwork.CreatedAt = timestamppb.New(time.Now())
	r.subnetworks = append(r.subnetworks, newSubnetwork)
	return newSubnetwork, nil
}

func (r *memorySubnetworkRepository) Delete(id uint32) error {
	for i, subnetwork := range r.subnetworks {
		if subnetwork.Id == id {
			r.subnetworks = append(r.subnetworks[:i], r.subnetworks[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("could not find subnetwork with id %d", id)
}

func (r *memorySubnetworkRepository) Update(id uint32, updateFn func(*pb.Subnetwork)) (*pb.Subnetwork, error) {
	for _, subnetwork := range r.subnetworks {
		if subnetwork.Id == id {
			updateFn(subnetwork)
			return subnetwork, nil
		}
	}

	return nil, fmt.Errorf("could not find subnetwork with id %d", id)
}
