package terraform_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSubnetworkResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "bx2cloud_subnetwork" "test" {
  cidr = "192.168.10.64/26"
}
`,
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
				Config: providerConfig + `
resource "bx2cloud_subnetwork" "test" {
  cidr = "192.168.10.192/26"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bx2cloud_subnetwork.test", "cidr", "192.168.10.192/26"),
				),
			},
		},
	})
}
