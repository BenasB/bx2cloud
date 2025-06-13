package shared

import "github.com/BenasB/bx2cloud/internal/api/pb"

type NetworkModel = pb.Network
type SubnetworkModel = pb.Subnetwork

type IpamType int

const (
	IPAM_UNALLOCATED IpamType = iota
	IPAM_CONTAINER
)
