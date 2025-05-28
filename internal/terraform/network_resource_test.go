package terraform_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNetworkResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "bx2cloud_network" "test" {
  internet_access = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("bx2cloud_network.test", "id"),
					resource.TestCheckResourceAttrSet("bx2cloud_network.test", "created_at"),
					resource.TestCheckResourceAttrSet("bx2cloud_network.test", "updated_at"),

					resource.TestCheckResourceAttr("bx2cloud_network.test", "internet_access", "true"),
				),
			},
			{
				ResourceName:            "bx2cloud_network.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"updated_at"},
			},
			{
				Config: providerConfig + `
resource "bx2cloud_network" "test" {
  internet_access = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bx2cloud_network.test", "internet_access", "false"),
				),
			},
		},
	})
}
