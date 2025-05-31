package network_test

import (
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

var testNetworks = []*shared.NetworkModel{
	&shared.NetworkModel{
		Id:             1,
		InternetAccess: false,
		CreatedAt:      timestamppb.New(time.Now().Add(-time.Hour)),
	},
	&shared.NetworkModel{
		Id:             42,
		InternetAccess: true,
		CreatedAt:      timestamppb.New(time.Now().Add(-time.Minute)),
	},
}

var mockConfigurator = network.NewMockConfigurator()

func TestNetwork_Create(t *testing.T) {
	repository := network.NewMemoryRepository(nil)
	subnetworkRepository := subnetwork.NewMemoryRepository(nil)
	service := network.NewService(repository, subnetworkRepository, mockConfigurator)
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
		repository := network.NewMemoryRepository(testNetworks)
		subnetworkRepository := subnetwork.NewMemoryRepository(nil)
		service := network.NewService(repository, subnetworkRepository, mockConfigurator)

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

func TestNetwork_Delete_SubnetworksExist(t *testing.T) {
	const subnetworksPerNetwork = 3
	testSubnetworks := make([]*pb.Subnetwork, len(testNetworks)*subnetworksPerNetwork)
	for i, n := range testNetworks {
		for j := 0; j < subnetworksPerNetwork; j++ {
			testSubnetworks[i*subnetworksPerNetwork+j] = &pb.Subnetwork{
				Id:        uint32(i*subnetworksPerNetwork + j),
				NetworkId: n.Id,
			}
		}
	}

	for _, tt := range testNetworks {
		repository := network.NewMemoryRepository(testNetworks)
		subnetworkRepository := subnetwork.NewMemoryRepository(testSubnetworks)
		service := network.NewService(repository, subnetworkRepository, mockConfigurator)

		t.Run(strconv.FormatUint(uint64(tt.Id), 10), func(t *testing.T) {
			_, err := service.Delete(t.Context(), &pb.NetworkIdentificationRequest{
				Id: tt.Id,
			})
			if err == nil || !strings.Contains(err.Error(), "still depends") {
				t.Error("Network was deleted even though it shouldn't have because a subnetwork depended on it")
			}
		})
	}
}

func TestNetwork_Delete_NetworkDoesNotExist(t *testing.T) {
	repository := network.NewMemoryRepository(nil)
	subnetworkRepository := subnetwork.NewMemoryRepository(nil)
	service := network.NewService(repository, subnetworkRepository, mockConfigurator)

	_, err := service.Delete(t.Context(), &pb.NetworkIdentificationRequest{
		Id: 1,
	})
	if err == nil {
		t.Error("There was no error returned by delete even though the network that we tried deleting does not exist")
	}
}

func TestNetwork_Get(t *testing.T) {
	for _, tt := range testNetworks {
		repository := network.NewMemoryRepository(testNetworks)
		subnetworkRepository := subnetwork.NewMemoryRepository(nil)
		service := network.NewService(repository, subnetworkRepository, mockConfigurator)

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
	stream := shared.NewMockStream[*shared.NetworkModel](t.Context())

	repository := network.NewMemoryRepository(testNetworks)
	subnetworkRepository := subnetwork.NewMemoryRepository(nil)
	service := network.NewService(repository, subnetworkRepository, mockConfigurator)
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
