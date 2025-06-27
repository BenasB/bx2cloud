package subnetwork_test

import (
	"encoding/binary"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BenasB/bx2cloud/internal/api/network"
	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"github.com/BenasB/bx2cloud/internal/api/subnetwork"
	"github.com/BenasB/bx2cloud/internal/api/subnetwork/ipam"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var testNetworks = []*shared.NetworkModel{
	&shared.NetworkModel{
		Id:             5123,
		InternetAccess: false,
		CreatedAt:      timestamppb.New(time.Now().Add(-time.Hour)),
	},
}

var testSubnetworks = []*shared.SubnetworkModel{
	&shared.SubnetworkModel{
		Id:           1,
		NetworkId:    testNetworks[0].Id,
		Address:      binary.BigEndian.Uint32([]byte{10, 0, 0, 0}),
		PrefixLength: 24,
		CreatedAt:    timestamppb.New(time.Now().Add(-time.Hour)),
	},
	&shared.SubnetworkModel{
		Id:           2,
		NetworkId:    testNetworks[0].Id,
		Address:      binary.BigEndian.Uint32([]byte{10, 0, 1, 0}),
		PrefixLength: 24,
		CreatedAt:    timestamppb.New(time.Now().Add(-time.Minute)),
	},
}

var mockConfigurator = subnetwork.NewMockConfigurator()

func TestSubnetwork_Create(t *testing.T) {
	repository := subnetwork.NewMemoryRepository(nil)
	networkRepository := network.NewMemoryRepository(testNetworks)
	ipamRepository := ipam.NewMemoryRepository()
	service := subnetwork.NewService(repository, networkRepository, mockConfigurator, ipamRepository)

	req := &pb.SubnetworkCreationRequest{
		NetworkId:    testNetworks[0].Id,
		Address:      binary.BigEndian.Uint32([]byte{192, 168, 0, 0}),
		PrefixLength: 30,
	}

	_, err := service.Create(t.Context(), req)
	if err != nil {
		t.Error(err)
	}
}

func TestSubnetwork_Create_NetworkNotFound(t *testing.T) {
	repository := subnetwork.NewMemoryRepository(nil)
	networkRepository := network.NewMemoryRepository(nil)
	ipamRepository := ipam.NewMemoryRepository()
	service := subnetwork.NewService(repository, networkRepository, mockConfigurator, ipamRepository)
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
		repository := subnetwork.NewMemoryRepository(testSubnetworks)
		networkRepository := network.NewMemoryRepository(testNetworks)
		ipamRepository := ipam.NewMemoryRepository()
		service := subnetwork.NewService(repository, networkRepository, mockConfigurator, ipamRepository)

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

func TestSubnetwork_Delete_StillAllocated(t *testing.T) {
	repository := subnetwork.NewMemoryRepository(testSubnetworks)
	networkRepository := network.NewMemoryRepository(testNetworks)
	sn, err := repository.Get(testSubnetworks[0].Id)
	if err != nil {
		t.Error(err)
	}
	ipamRepository := ipam.NewMemoryRepository()
	if _, err := ipamRepository.Allocate(sn, shared.IPAM_CONTAINER); err != nil {
		t.Error(err)
	}

	service := subnetwork.NewService(repository, networkRepository, mockConfigurator, ipamRepository)
	_, err = service.Delete(t.Context(), &pb.SubnetworkIdentificationRequest{
		Id: sn.Id,
	})
	if err == nil || !strings.Contains(err.Error(), "allocated") {
		t.Error("Subnetwork was deleted even though it had at least 1 resource IP allocated")
	}
}

func TestSubnetwork_Get(t *testing.T) {
	for _, tt := range testSubnetworks {
		repository := subnetwork.NewMemoryRepository(testSubnetworks)
		networkRepository := network.NewMemoryRepository(nil)
		ipamRepository := ipam.NewMemoryRepository()
		service := subnetwork.NewService(repository, networkRepository, mockConfigurator, ipamRepository)

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
	stream := shared.NewMockStream[*shared.SubnetworkModel](t.Context())

	repository := subnetwork.NewMemoryRepository(testSubnetworks)
	networkRepository := network.NewMemoryRepository(nil)
	ipamRepository := ipam.NewMemoryRepository()
	service := subnetwork.NewService(repository, networkRepository, mockConfigurator, ipamRepository)
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
