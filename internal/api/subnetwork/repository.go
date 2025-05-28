package subnetwork

import (
	"context"
	"fmt"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/id"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type subnetworkRepository interface {
	get(id uint32) (*pb.Subnetwork, error)
	getAll(ctx context.Context) (<-chan *pb.Subnetwork, <-chan error)
	add(subnetwork *pb.Subnetwork) (*pb.Subnetwork, error)
	delete(id uint32) error
	update(id uint32, updateFn func(*pb.Subnetwork)) (*pb.Subnetwork, error)
}

var _ subnetworkRepository = &memorySubnetworkRepository{}

// Caution: not thread safe
type memorySubnetworkRepository struct {
	subnetworks []*pb.Subnetwork
}

func NewMemorySubnetworkRepository(subnetworks []*pb.Subnetwork) subnetworkRepository {
	sns := make([]*pb.Subnetwork, len(subnetworks))
	for i, subnetwork := range subnetworks {
		sns[i] = proto.Clone(subnetwork).(*pb.Subnetwork)
	}

	return &memorySubnetworkRepository{
		subnetworks: sns,
	}
}

func (r *memorySubnetworkRepository) get(id uint32) (*pb.Subnetwork, error) {
	for _, subnetwork := range r.subnetworks {
		if subnetwork.Id == id {
			return subnetwork, nil
		}
	}

	return nil, fmt.Errorf("could not find subnetwork")
}

func (r *memorySubnetworkRepository) getAll(ctx context.Context) (<-chan *pb.Subnetwork, <-chan error) {
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

func (r *memorySubnetworkRepository) add(subnetwork *pb.Subnetwork) (*pb.Subnetwork, error) {
	//newSubnetwork := *subnetwork
	newSubnetwork := proto.Clone(subnetwork).(*pb.Subnetwork)
	newSubnetwork.Id = id.NextId("subnetwork")
	newSubnetwork.CreatedAt = timestamppb.New(time.Now())
	r.subnetworks = append(r.subnetworks, newSubnetwork)
	return newSubnetwork, nil
}

func (r *memorySubnetworkRepository) delete(id uint32) error {
	for i, subnetwork := range r.subnetworks {
		if subnetwork.Id == id {
			r.subnetworks = append(r.subnetworks[:i], r.subnetworks[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("could not find subnetwork")
}

func (r *memorySubnetworkRepository) update(id uint32, updateFn func(*pb.Subnetwork)) (*pb.Subnetwork, error) {
	for _, subnetwork := range r.subnetworks {
		if subnetwork.Id == id {
			updateFn(subnetwork)
			return subnetwork, nil
		}
	}

	return nil, fmt.Errorf("could not find subnetwork")
}
