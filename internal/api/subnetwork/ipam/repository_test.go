package ipam_test

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/BenasB/bx2cloud/internal/api/interfaces"
	"github.com/BenasB/bx2cloud/internal/api/subnetwork/ipam"
)

func TestIpam_Memory_Allocate(t *testing.T) {
	repository := ipam.NewMemoryRepository()
	subnetwork := &interfaces.SubnetworkModel{
		Id:           1,
		Address:      binary.BigEndian.Uint32([]byte{10, 0, 42, 0}),
		PrefixLength: 24,
	}

	ip, err := repository.Allocate(subnetwork, interfaces.IPAM_CONTAINER)
	if err != nil {
		t.Error(err)
	}

	if !ip.IP.Equal(net.IPv4(10, 0, 42, 2)) {
		t.Error("First allocated IP was supposed to be 10.0.42.2")
	}
}

func TestIpam_Memory_Dogfood(t *testing.T) {
	repository := ipam.NewMemoryRepository()
	subnetwork := &interfaces.SubnetworkModel{
		Id:           1,
		Address:      binary.BigEndian.Uint32([]byte{10, 0, 42, 0}),
		PrefixLength: 24,
	}

	ip, err := repository.Allocate(subnetwork, interfaces.IPAM_CONTAINER)
	if err != nil {
		t.Error(err)
	}

	err = repository.Deallocate(subnetwork, ip)
	if err != nil {
		t.Error(err)
	}
}
