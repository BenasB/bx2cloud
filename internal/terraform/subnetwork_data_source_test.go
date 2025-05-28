package terraform_test

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSubnetworkDataSource(t *testing.T) {
	createReq := &pb.SubnetworkCreationRequest{
		Address:      3232238088, // 192.168.10.8
		PrefixLength: 30,
	}
	subnetwork, err := grpcClients.Subnetwork.Create(t.Context(), createReq)
	if err != nil {
		t.Fatalf("Failed to create a subnetwork before running the terraform test: %v", err)
	}

	t.Cleanup(func() {
		deleteReq := &pb.SubnetworkIdentificationRequest{
			Id: subnetwork.Id,
		}
		_, err = grpcClients.Subnetwork.Delete(context.Background(), deleteReq)
		if err != nil {
			t.Fatalf("Failed to delete subnetwork '%d' after running the terraform test: %v", subnetwork.Id, err)
		}
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: providerConfig + "data \"bx2cloud_subnetwork\" \"test\" {\n" +
					fmt.Sprintf("id = %d\n", subnetwork.Id) +
					"}",
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.bx2cloud_subnetwork.test", "cidr", "192.168.10.8/30"),
				),
			},
		},
	})
}
