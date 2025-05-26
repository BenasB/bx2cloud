package handlers

import (
	"strconv"
	"testing"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var testNetworks = []*pb.Network{
	&pb.Network{
		Id:             1,
		InternetAccess: false,
		CreatedAt:      timestamppb.New(time.Now().Add(-time.Hour)),
	},
	&pb.Network{
		Id:             2,
		InternetAccess: true,
		CreatedAt:      timestamppb.New(time.Now().Add(-time.Minute)),
	},
}

type mockStream[T any] struct {
	grpc.ServerStream
	SentItems []T
}

func (s *mockStream[T]) Send(item T) error {
	s.SentItems = append(s.SentItems, item)
	return nil
}

func TestNetwork_Create(t *testing.T) {
	service := NewNetworkService(make([]*pb.Network, 0))
	req := &pb.NetworkCreationRequest{
		InternetAccess: true,
	}

	_, err := service.Create(t.Context(), req)
	if err != nil {
		t.Error(err)
	}
}

func TestNetwork_Delete(t *testing.T) {
	for _, tt := range testNetworks {
		service := NewNetworkService(testNetworks)

		t.Run(strconv.FormatUint(uint64(tt.Id), 10), func(t *testing.T) {
			_, err := service.Delete(t.Context(), &pb.NetworkIdentificationRequest{
				Id: tt.Id,
			})
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestNetwork_Get(t *testing.T) {
	for _, tt := range testNetworks {
		service := NewNetworkService(testNetworks)

		t.Run(strconv.FormatUint(uint64(tt.Id), 10), func(t *testing.T) {
			resp, err := service.Get(t.Context(), &pb.NetworkIdentificationRequest{
				Id: tt.Id,
			})
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(tt, resp, protocmp.Transform()); diff != "" {
				t.Errorf("network mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNetwork_List(t *testing.T) {
	stream := &mockStream[*pb.Network]{}

	service := NewNetworkService(testNetworks)
	service.List(&emptypb.Empty{}, stream)

	if len(testNetworks) != len(stream.SentItems) {
		t.Error("not the same amount of networks received")
	}

	for i := 0; i < len(testNetworks); i++ {
		if diff := cmp.Diff(testNetworks[i], stream.SentItems[i], protocmp.Transform()); diff != "" {
			t.Errorf("network mismatch (-want +got):\n%s", diff)
		}
	}
}
