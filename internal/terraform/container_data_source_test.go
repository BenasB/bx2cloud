package terraform_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccContainerDataSource(t *testing.T) {
	networkCreateReq := &pb.NetworkCreationRequest{
		InternetAccess: true,
	}

	network, err := grpcClients.Network.Create(t.Context(), networkCreateReq)
	if err != nil {
		t.Fatalf("Failed to create a network before running the terraform test: %v", err)
	}

	subnetworkCreateReq := &pb.SubnetworkCreationRequest{
		NetworkId:    network.Id,
		Address:      binary.BigEndian.Uint32([]byte{192, 168, 42, 0}),
		PrefixLength: 24,
	}

	subnetwork, err := grpcClients.Subnetwork.Create(t.Context(), subnetworkCreateReq)
	if err != nil {
		t.Fatalf("Failed to create a subnetwork before running the terraform test: %v", err)
	}

	containerCreateReq := &pb.ContainerCreationRequest{
		SubnetworkId: subnetwork.Id,
		Image:        "ubuntu:24.04",
		Entrypoint:   []string{"/bin/sh"},
		Cmd:          []string{"sleep", "infinity"},
	}
	container, err := grpcClients.Container.Create(t.Context(), containerCreateReq)
	if err != nil {
		t.Fatalf("Failed to create a container before running the terraform test: %v", err)
	}

	t.Cleanup(func() {
		containerDeleteReq := &pb.ContainerIdentificationRequest{
			Id: container.Id,
		}
		_, err = grpcClients.Container.Delete(context.Background(), containerDeleteReq)
		if err != nil {
			t.Fatalf("Failed to delete container '%d' after running the terraform test: %v", container.Id, err)
		}

		subnetworkDeleteReq := &pb.SubnetworkIdentificationRequest{
			Id: subnetwork.Id,
		}

		_, err = grpcClients.Subnetwork.Delete(context.Background(), subnetworkDeleteReq)
		if err != nil {
			t.Fatalf("Failed to delete subnetwork '%d' after running the terraform test: %v", subnetwork.Id, err)
		}

		networkDeleteReq := &pb.NetworkIdentificationRequest{
			Id: network.Id,
		}
		_, err = grpcClients.Network.Delete(context.Background(), networkDeleteReq)
		if err != nil {
			t.Fatalf("Failed to delete network '%d' after running the terraform test: %v", network.Id, err)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(providerConfig+`
data "bx2cloud_container" "test" {
  id = %d
}`, container.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bx2cloud_container.test", "image", "ubuntu:24.04"),
					resource.TestCheckResourceAttrSet("data.bx2cloud_container.test", "status"),
					resource.TestCheckResourceAttrSet("data.bx2cloud_container.test", "ip"),
				),
			},
		},
	})
}
