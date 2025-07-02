package ipam

import (
	"fmt"
	"math"
	"net"

	"github.com/BenasB/bx2cloud/internal/api/interfaces"
)

var _ interfaces.IpamRepository = &memoryRepository{}

// Caution: not thread safe
type memoryRepository struct {
	subnetworkAllocations map[uint32][]interfaces.IpamType
	reservedIpCount       uint32
}

func NewMemoryRepository() interfaces.IpamRepository {
	return &memoryRepository{
		subnetworkAllocations: make(map[uint32][]interfaces.IpamType),
		reservedIpCount:       1,
	}
}

func (r *memoryRepository) Allocate(subnetwork *interfaces.SubnetworkModel, resourceType interfaces.IpamType) (*net.IPNet, error) {
	allocations, exists := r.subnetworkAllocations[subnetwork.Id]

	if !exists {
		allocations = r.initSubnetworkAllocation(subnetwork)
	}

	for i := range allocations {
		if allocations[i] != interfaces.IPAM_UNALLOCATED {
			continue
		}

		allocations[i] = resourceType
		ip := subnetwork.Address + r.reservedIpCount + 1 + uint32(i)
		return &net.IPNet{
			IP:   net.IPv4(byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip)).To4(),
			Mask: net.CIDRMask(int(subnetwork.PrefixLength), 32),
		}, nil
	}

	return nil, fmt.Errorf("subnetwork has run out of allocatable IPs")
}

func (r *memoryRepository) Deallocate(subnetwork *interfaces.SubnetworkModel, ip *net.IPNet) error {
	allocations, exists := r.subnetworkAllocations[subnetwork.Id]

	if !exists {
		return fmt.Errorf("subnetwork does not have this IP allocated")
	}

	address := uint32(ip.IP[0])<<24 | uint32(ip.IP[1])<<16 | uint32(ip.IP[2])<<8 | uint32(ip.IP[3])
	i := address - subnetwork.Address - r.reservedIpCount - 1
	if i < 0 || i >= uint32(len(allocations)) {
		return fmt.Errorf("IP is outside of bounds of the subnetwork")
	}

	if allocations[i] == interfaces.IPAM_UNALLOCATED {
		return fmt.Errorf("subnetwork does not have this IP allocated")
	}

	allocations[i] = interfaces.IPAM_UNALLOCATED
	return nil
}

func (r *memoryRepository) HasAllocations(subnetwork *interfaces.SubnetworkModel) (interfaces.IpamType, bool) {
	allocations, exists := r.subnetworkAllocations[subnetwork.Id]

	if !exists {
		return interfaces.IPAM_UNALLOCATED, false
	}

	for i := range allocations {
		if allocations[i] != interfaces.IPAM_UNALLOCATED {
			return allocations[i], true
		}
	}

	return interfaces.IPAM_UNALLOCATED, false
}

func (r *memoryRepository) GetSubnetworkGateway(subnetwork *interfaces.SubnetworkModel) *net.IPNet {
	ip := subnetwork.Address + 1

	return &net.IPNet{
		IP:   net.IPv4(byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip)),
		Mask: net.CIDRMask(int(subnetwork.PrefixLength), 32),
	}
}

func (r *memoryRepository) initSubnetworkAllocation(subnetwork *interfaces.SubnetworkModel) []interfaces.IpamType {
	noOfHosts := uint32(math.Pow(2, float64(32-subnetwork.PrefixLength))) - 2
	unreservedNoOfHosts := noOfHosts - r.reservedIpCount

	allocations := make([]interfaces.IpamType, unreservedNoOfHosts)

	r.subnetworkAllocations[subnetwork.Id] = allocations
	return allocations
}
