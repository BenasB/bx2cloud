package terraform_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSubnetworkDataSource(t *testing.T) {
	networkCreateReq := &pb.NetworkCreationRequest{
		InternetAccess: true,
	}

	network, err := grpcClients.Network.Create(t.Context(), networkCreateReq)
	if err != nil {
		t.Fatalf("Failed to create a network before running the terraform test: %v", err)
	}

	subnetworkCreateReq := &pb.SubnetworkCreationRequest{
		NetworkId:    network.Id,
		Address:      3232238088, // 192.168.10.8
		PrefixLength: 30,
	}
	subnetwork, err := grpcClients.Subnetwork.Create(t.Context(), subnetworkCreateReq)
	if err != nil {
		t.Fatalf("Failed to create a subnetwork before running the terraform test: %v", err)
	}

	t.Cleanup(func() {
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
data "bx2cloud_subnetwork" "test" {
  id = %d
}`, subnetwork.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bx2cloud_subnetwork.test", "cidr", "192.168.10.8/30"),
				),
			},
		},
	})
}
