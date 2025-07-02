package terraform_test

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccContainerResource(t *testing.T) {
	networkCreateReq := &pb.NetworkCreationRequest{
		InternetAccess: true,
	}

	network, err := grpcClients.Network.Create(t.Context(), networkCreateReq)
	if err != nil {
		t.Fatalf("Failed to create a network before running the terraform test: %v", err)
	}

	subnetworkCreateReq := &pb.SubnetworkCreationRequest{
		NetworkId:    network.Id,
		Address:      binary.BigEndian.Uint32([]byte{192, 168, 43, 0}),
		PrefixLength: 24,
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
resource "bx2cloud_container" "test" {
  subnetwork_id = %d
  image = "ubuntu:24.04"
  entrypoint = ["/bin/sh", "-c"]
  cmd = ["sleep infinity"]
  env = {
    FOO = "bar"
  }
}`, subnetwork.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bx2cloud_container.test", "id"),
					resource.TestCheckResourceAttrSet("bx2cloud_container.test", "started_at"),
					resource.TestCheckResourceAttrSet("bx2cloud_container.test", "created_at"),
					resource.TestCheckResourceAttrSet("bx2cloud_container.test", "updated_at"),

					resource.TestCheckResourceAttr("bx2cloud_container.test", "status", "running"),
					resource.TestCheckResourceAttrWith("bx2cloud_container.test", "ip", func(value string) error {
						if !strings.HasPrefix(value, "192.168.43.") {
							return fmt.Errorf("the container's ip is not in the expected subnetwork")
						}

						return nil
					}),
				),
			},
			{
				ResourceName:            "bx2cloud_container.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
			},
			{
				Config: fmt.Sprintf(providerConfig+`
resource "bx2cloud_container" "test" {
  subnetwork_id = %d
  image = "ubuntu:24.04"
  entrypoint = ["/bin/sh", "-c"]
  cmd = ["sleep infinity"]
  env = {
    FOO = "bar"
  }
  status = "stopped"
}`, subnetwork.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bx2cloud_container.test", "status", "stopped"),
				),
			},
			{
				Config: fmt.Sprintf(providerConfig+`
resource "bx2cloud_container" "test" {
  subnetwork_id = %d
  image = "ubuntu:24.04"
  entrypoint = ["/bin/sh", "-c"]
  cmd = ["sleep infinity"]
  env = {
    FOO = "bar"
  }
  status = "running"
}`, subnetwork.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bx2cloud_container.test", "status", "running"),
				),
			},
			{
				Config: fmt.Sprintf(providerConfig+`
resource "bx2cloud_container" "test" {
  subnetwork_id = %d
  image = "ubuntu:22.04"
  entrypoint = ["/bin/sh", "-c"]
  cmd = ["sleep infinity"]
  env = {
    FOO = "bar"
  }
  status = "running"
}`, subnetwork.Id),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("bx2cloud_container.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
			{
				Config: fmt.Sprintf(providerConfig+`
resource "bx2cloud_container" "test" {
  subnetwork_id = %d
  image = "ubuntu:22.04"
  entrypoint = ["/bin/../bin/sh", "-c"]
  cmd = ["sleep infinity"]
  env = {
    FOO = "bar"
  }
  status = "running"
}`, subnetwork.Id),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("bx2cloud_container.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
			{
				Config: fmt.Sprintf(providerConfig+`
resource "bx2cloud_container" "test" {
  subnetwork_id = %d
  image = "ubuntu:22.04"
  entrypoint = ["/bin/../bin/sh", "-c"]
  cmd = ["sleep 1000"]
  env = {
    FOO = "bar"
  }
  status = "running"
}`, subnetwork.Id),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("bx2cloud_container.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
			{
				Config: fmt.Sprintf(providerConfig+`
resource "bx2cloud_container" "test" {
  subnetwork_id = %d
  image = "ubuntu:22.04"
  entrypoint = ["/bin/../bin/sh", "-c"]
  cmd = ["sleep 1000"]
  env = {
    FOO = "bar"
	DIFF = "new"
  }
  status = "running"
}`, subnetwork.Id),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("bx2cloud_container.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
		},
	})
}
