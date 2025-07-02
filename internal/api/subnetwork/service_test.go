package subnetwork_test

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BenasB/bx2cloud/internal/api/interfaces"
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

var testNetworks = []*interfaces.NetworkModel{
	&interfaces.NetworkModel{
		Id:             5123,
		InternetAccess: false,
		CreatedAt:      timestamppb.New(time.Now().Add(-time.Hour)),
	},
}

var testSubnetworks = []*interfaces.SubnetworkModel{
	&interfaces.SubnetworkModel{
		Id:           1,
		NetworkId:    testNetworks[0].Id,
		Address:      binary.BigEndian.Uint32([]byte{10, 0, 0, 0}),
		PrefixLength: 24,
		CreatedAt:    timestamppb.New(time.Now().Add(-time.Hour)),
	},
	&interfaces.SubnetworkModel{
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
	if err == nil {
		t.Error("Subnetwork was created even though the network it is supposed to depend on does not exist")
	} else if !strings.Contains(err.Error(), "could not find network") {
		t.Errorf("Subnetwork was not created but not because the network it is supposed to depend on does not exist: %v", err)
	}
}

func TestSubnetwork_Create_Overlap(t *testing.T) {
	tests := []struct {
		existing *net.IPNet
		new      *net.IPNet
	}{
		{
			existing: &net.IPNet{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(24, 32)},
			new:      &net.IPNet{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(16, 32)},
		},
		{
			existing: &net.IPNet{IP: net.IPv4(10, 0, 1, 0), Mask: net.CIDRMask(24, 32)},
			new:      &net.IPNet{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(16, 32)},
		},
		{
			existing: &net.IPNet{IP: net.IPv4(10, 0, 255, 0), Mask: net.CIDRMask(24, 32)},
			new:      &net.IPNet{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(16, 32)},
		},
		{
			existing: &net.IPNet{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(16, 32)},
			new:      &net.IPNet{IP: net.IPv4(10, 0, 1, 0), Mask: net.CIDRMask(24, 32)},
		},
		{
			existing: &net.IPNet{IP: net.IPv4(10, 0, 4, 0), Mask: net.CIDRMask(24, 32)},
			new:      &net.IPNet{IP: net.IPv4(10, 0, 12, 0), Mask: net.CIDRMask(23, 32)},
		},
	}

	for _, tt := range tests {
		existingPrefixLength, _ := tt.existing.Mask.Size()
		existingSubnetwork := &interfaces.SubnetworkModel{
			Id:           1,
			NetworkId:    testNetworks[0].Id,
			Address:      binary.BigEndian.Uint32(tt.existing.IP),
			PrefixLength: uint32(existingPrefixLength),
			CreatedAt:    timestamppb.New(time.Now().Add(-time.Hour)),
		}
		repository := subnetwork.NewMemoryRepository([]*interfaces.SubnetworkModel{existingSubnetwork})
		networkRepository := network.NewMemoryRepository(testNetworks)
		ipamRepository := ipam.NewMemoryRepository()
		service := subnetwork.NewService(repository, networkRepository, mockConfigurator, ipamRepository)

		t.Run(fmt.Sprintf("%s:%s", tt.existing.String(), tt.new.String()), func(t *testing.T) {
			newPrefixLength, _ := tt.existing.Mask.Size()
			req := &pb.SubnetworkCreationRequest{
				NetworkId:    testNetworks[0].Id,
				Address:      binary.BigEndian.Uint32(tt.new.IP),
				PrefixLength: uint32(newPrefixLength),
			}

			_, err := service.Create(t.Context(), req)
			if err == nil {
				t.Error("Subnetwork was created even though it overlaps with other subnetwork in the same network")
			} else if !strings.Contains(err.Error(), "overlap") {
				t.Errorf("Subnetwork was created even though it overlaps with other subnetwork in the same network: %v", err)
			}
		})
	}
}

func TestSubnetwork_Create_NonOverlap(t *testing.T) {
	existingSubnetwork := &interfaces.SubnetworkModel{
		Id:           1,
		NetworkId:    testNetworks[0].Id,
		Address:      binary.BigEndian.Uint32([]byte{10, 0, 42, 0}),
		PrefixLength: 24,
		CreatedAt:    timestamppb.New(time.Now().Add(-time.Hour)),
	}
	repository := subnetwork.NewMemoryRepository([]*interfaces.SubnetworkModel{existingSubnetwork})
	networkRepository := network.NewMemoryRepository(testNetworks)
	ipamRepository := ipam.NewMemoryRepository()
	service := subnetwork.NewService(repository, networkRepository, mockConfigurator, ipamRepository)

	req := &pb.SubnetworkCreationRequest{
		NetworkId:    testNetworks[0].Id,
		Address:      binary.BigEndian.Uint32([]byte{10, 0, 43, 0}),
		PrefixLength: 24,
	}

	_, err := service.Create(t.Context(), req)
	if err != nil {
		t.Error(err)
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
	if _, err := ipamRepository.Allocate(sn, interfaces.IPAM_CONTAINER); err != nil {
		t.Error(err)
	}

	service := subnetwork.NewService(repository, networkRepository, mockConfigurator, ipamRepository)
	_, err = service.Delete(t.Context(), &pb.SubnetworkIdentificationRequest{
		Id: sn.Id,
	})
	if err == nil {
		t.Error("Subnetwork was deleted even though it had at least 1 resource IP allocated")
	} else if !strings.Contains(err.Error(), "allocated") {
		t.Errorf("Subnetwork was not deleted, but not due to having at least 1 resource IP allocated: %v", err)
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
	stream := shared.NewMockStream[*interfaces.SubnetworkModel](t.Context())

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
