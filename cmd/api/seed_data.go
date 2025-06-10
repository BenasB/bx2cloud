package main

import (
	"encoding/binary"
	"time"

	"github.com/BenasB/bx2cloud/internal/api/id"
	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/BenasB/bx2cloud/internal/api/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var sampleNetworks = []*shared.NetworkModel{
	&pb.Network{
		Id:             id.NextId("network"),
		InternetAccess: false,
		CreatedAt:      timestamppb.New(time.Now().Add(-time.Hour)),
	},
	&pb.Network{
		Id:             id.NextId("network"),
		InternetAccess: true,
		CreatedAt:      timestamppb.New(time.Now().Add(-time.Minute)),
	},
	&pb.Network{
		Id:             id.NextId("network"),
		InternetAccess: true,
		CreatedAt:      timestamppb.New(time.Now().Add(-time.Minute * 30)),
	},
}

var sampleSubnetworks = []*shared.SubnetworkModel{
	&pb.Subnetwork{
		Id:           id.NextId("subnetwork"),
		NetworkId:    sampleNetworks[0].Id,
		Address:      binary.BigEndian.Uint32([]byte{10, 0, 0, 0}),
		PrefixLength: 24,
		CreatedAt:    timestamppb.New(time.Now().Add(-time.Hour)),
	},
	&pb.Subnetwork{
		Id:           id.NextId("subnetwork"),
		NetworkId:    sampleNetworks[0].Id,
		Address:      binary.BigEndian.Uint32([]byte{10, 0, 1, 0}),
		PrefixLength: 24,
		CreatedAt:    timestamppb.New(time.Now().Add(-time.Minute)),
	},
	&pb.Subnetwork{
		Id:           id.NextId("subnetwork"),
		NetworkId:    sampleNetworks[2].Id,
		Address:      binary.BigEndian.Uint32([]byte{192, 168, 0, 64}),
		PrefixLength: 26,
		CreatedAt:    timestamppb.New(time.Now().Add(-time.Minute * 29)),
	},
}
