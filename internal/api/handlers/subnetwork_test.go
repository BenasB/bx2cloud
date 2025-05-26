package handlers

import (
	"encoding/binary"
	"strconv"
	"testing"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
	service := NewSubnetworkService(make([]*pb.Subnetwork, 0))
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
		service := NewSubnetworkService(testSubnetworks)

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
		service := NewSubnetworkService(testSubnetworks)

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
	stream := &mockStream[*pb.Subnetwork]{}

	service := NewSubnetworkService(testSubnetworks)
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
