package terraform_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNetworkDataSource(t *testing.T) {
	createReq := &pb.NetworkCreationRequest{
		InternetAccess: true,
	}
	network, err := grpcClients.Network.Create(t.Context(), createReq)
	if err != nil {
		t.Fatalf("Failed to create a network before running the terraform test: %v", err)
	}

	t.Cleanup(func() {
		deleteReq := &pb.NetworkIdentificationRequest{
			Id: network.Id,
		}
		_, err = grpcClients.Network.Delete(context.Background(), deleteReq)
		if err != nil {
			t.Fatalf("Failed to delete network '%d' after running the terraform test: %v", network.Id, err)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(providerConfig+`
data "bx2cloud_network" "test" {
  id = %d
}`, network.Id),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bx2cloud_network.test", "internet_access", strconv.FormatBool(createReq.InternetAccess)),
				),
			},
		},
	})
}
