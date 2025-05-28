package subnetwork_test

import (
	"context"
	"encoding/binary"
	"strconv"
	"testing"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/subnetwork"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mockStream[T any] struct {
	grpc.ServerStream
	SentItems []T
	ctx       context.Context
}

func (s *mockStream[T]) Send(item T) error {
	s.SentItems = append(s.SentItems, item)
	return nil
}

func (s *mockStream[T]) Context() context.Context {
	return s.ctx
}

var testSubnetworks = []*pb.Subnetwork{
	&pb.Subnetwork{
		Id:           1,
		Address:      binary.BigEndian.Uint32([]byte{10, 0, 0, 0}),
		PrefixLength: 24,
		CreatedAt:    timestamppb.New(time.Now().Add(-time.Hour)),
	},
	&pb.Subnetwork{
		Id:           2,
		Address:      binary.BigEndian.Uint32([]byte{10, 0, 1, 0}),
		PrefixLength: 24,
		CreatedAt:    timestamppb.New(time.Now().Add(-time.Minute)),
	},
}

func TestSubnetwork_Create(t *testing.T) {
	repository := subnetwork.NewMemorySubnetworkRepository(make([]*pb.Subnetwork, 0))
	service := subnetwork.NewSubnetworkService(repository)
	req := &pb.SubnetworkCreationRequest{
		Address:      binary.BigEndian.Uint32([]byte{192, 168, 0, 0}),
		PrefixLength: 30,
	}

	_, err := service.Create(t.Context(), req)
	if err != nil {
		t.Error(err)
	}
}

func TestSubnetwork_Delete(t *testing.T) {
	for _, tt := range testSubnetworks {
		repository := subnetwork.NewMemorySubnetworkRepository(testSubnetworks)
		service := subnetwork.NewSubnetworkService(repository)

		t.Run(strconv.FormatUint(uint64(tt.Id), 10), func(t *testing.T) {
			_, err := service.Delete(t.Context(), &pb.SubnetworkIdentificationRequest{
				Id: tt.Id,
			})
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestSubnetwork_Get(t *testing.T) {
	for _, tt := range testSubnetworks {
		repository := subnetwork.NewMemorySubnetworkRepository(testSubnetworks)
		service := subnetwork.NewSubnetworkService(repository)

		t.Run(strconv.FormatUint(uint64(tt.Id), 10), func(t *testing.T) {
			resp, err := service.Get(t.Context(), &pb.SubnetworkIdentificationRequest{
				Id: tt.Id,
			})
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(tt, resp, protocmp.Transform()); diff != "" {
				t.Errorf("subnetwork mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSubnetwork_List(t *testing.T) {
	stream := &mockStream[*pb.Subnetwork]{
		ctx: t.Context(),
	}

	repository := subnetwork.NewMemorySubnetworkRepository(testSubnetworks)
	service := subnetwork.NewSubnetworkService(repository)
	service.List(&emptypb.Empty{}, stream)

	if len(testSubnetworks) != len(stream.SentItems) {
		t.Error("not the same amount of subnetworks received")
	}

	for i := 0; i < len(testSubnetworks); i++ {
		if diff := cmp.Diff(testSubnetworks[i], stream.SentItems[i], protocmp.Transform()); diff != "" {
			t.Errorf("subnetwork mismatch (-want +got):\n%s", diff)
		}
	}
}
