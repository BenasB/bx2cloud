package terraform_test

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccSubnetworkResource(t *testing.T) {
	networkOneCreateReq := &pb.NetworkCreationRequest{
		InternetAccess: true,
	}

	networkOne, err := grpcClients.Network.Create(t.Context(), networkOneCreateReq)
	if err != nil {
		t.Fatalf("Failed to create a network before running the terraform test: %v", err)
	}

	networkTwoCreateReq := &pb.NetworkCreationRequest{
		InternetAccess: false,
	}

	networkTwo, err := grpcClients.Network.Create(t.Context(), networkTwoCreateReq)
	if err != nil {
		t.Fatalf("Failed to create a network before running the terraform test: %v", err)
	}

	t.Cleanup(func() {
		networkOneDeleteReq := &pb.NetworkIdentificationRequest{
			Id: networkOne.Id,
		}
		_, err = grpcClients.Network.Delete(context.Background(), networkOneDeleteReq)
		if err != nil {
			t.Fatalf("Failed to delete network '%d' after running the terraform test: %v", networkOne.Id, err)
		}
		networkTwoDeleteReq := &pb.NetworkIdentificationRequest{
			Id: networkOne.Id,
		}
		_, err = grpcClients.Network.Delete(context.Background(), networkTwoDeleteReq)
		if err != nil {
			t.Fatalf("Failed to delete network '%d' after running the terraform test: %v", networkOne.Id, err)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(providerConfig+`
resource "bx2cloud_subnetwork" "test" {
  cidr = "192.168.10.64/26"
  network_id = %d
}`, networkOne.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bx2cloud_subnetwork.test", "id"),
					resource.TestCheckResourceAttrSet("bx2cloud_subnetwork.test", "created_at"),
					resource.TestCheckResourceAttrSet("bx2cloud_subnetwork.test", "updated_at"),

					resource.TestCheckResourceAttr("bx2cloud_subnetwork.test", "cidr", "192.168.10.64/26"),
				),
			},
			{
				ResourceName:            "bx2cloud_subnetwork.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
			},
			{
				Config: fmt.Sprintf(providerConfig+`
resource "bx2cloud_subnetwork" "test" {
  cidr = "192.168.10.192/26"
  network_id = %d
}`, networkOne.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bx2cloud_subnetwork.test", "cidr", "192.168.10.192/26"),
				),
			},
			{
				Config: fmt.Sprintf(providerConfig+`
resource "bx2cloud_subnetwork" "test" {
  cidr = "192.168.10.192/26"
  network_id = %d
}`, networkTwo.Id),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("bx2cloud_subnetwork.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
		},
	})
}
