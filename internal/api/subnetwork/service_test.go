package subnetwork_test

import (
	"encoding/binary"
	"strconv"
	"strings"
	"testing"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/BenasB/bx2cloud/internal/api/network"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"github.com/BenasB/bx2cloud/internal/api/subnetwork"
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
	repository := subnetwork.NewMemorySubnetworkRepository(make([]*pb.Subnetwork, 0))
	testNetwork := &pb.Network{
		Id:             42,
		InternetAccess: true,
		CreatedAt:      timestamppb.New(time.Now().Add(-time.Hour)),
	}
	networkRepository := network.NewMemoryNetworkRepository([]*pb.Network{testNetwork})
	service := subnetwork.NewSubnetworkService(repository, networkRepository)

	req := &pb.SubnetworkCreationRequest{
		NetworkId:    testNetwork.Id,
		Address:      binary.BigEndian.Uint32([]byte{192, 168, 0, 0}),
		PrefixLength: 30,
	}

	_, err := service.Create(t.Context(), req)
	if err != nil {
		t.Error(err)
	}
}

func TestSubnetwork_Create_NetworkNotFound(t *testing.T) {
	repository := subnetwork.NewMemorySubnetworkRepository(make([]*pb.Subnetwork, 0))
	networkRepository := network.NewMemoryNetworkRepository(nil)
	service := subnetwork.NewSubnetworkService(repository, networkRepository)
	req := &pb.SubnetworkCreationRequest{
		NetworkId:    0,
		Address:      binary.BigEndian.Uint32([]byte{192, 168, 0, 0}),
		PrefixLength: 30,
	}

	_, err := service.Create(t.Context(), req)
	if err == nil || !strings.Contains(err.Error(), "could not find network") {
		t.Error("Subnetwork was created even though the network it is supposed to depend on does not exist")
	}
}

func TestSubnetwork_Delete(t *testing.T) {
	for _, tt := range testSubnetworks {
		repository := subnetwork.NewMemorySubnetworkRepository(testSubnetworks)
		networkRepository := network.NewMemoryNetworkRepository(nil)
		service := subnetwork.NewSubnetworkService(repository, networkRepository)

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
		networkRepository := network.NewMemoryNetworkRepository(nil)
		service := subnetwork.NewSubnetworkService(repository, networkRepository)

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
	stream := shared.NewMockStream[*pb.Subnetwork](t.Context())

	repository := subnetwork.NewMemorySubnetworkRepository(testSubnetworks)
	networkRepository := network.NewMemoryNetworkRepository(nil)
	service := subnetwork.NewSubnetworkService(repository, networkRepository)
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
