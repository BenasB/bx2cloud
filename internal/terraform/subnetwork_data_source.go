package terraform

import (
	"context"
	"fmt"
	"strconv"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &subnetworkDataSource{}
	_ datasource.DataSourceWithConfigure = &subnetworkDataSource{}
)

func NewSubnetworkDataSource() datasource.DataSource {
	return &subnetworkDataSource{}
}

type subnetworkDataSource struct {
	client pb.SubnetworkServiceClient
}

type subnetworkDataSourceModel struct {
	Id        types.Int32  `tfsdk:"id"`
	Cidr      types.String `tfsdk:"cidr"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (d *subnetworkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*bx2cloudClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *bx2cloudClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = clients.subnetwork
}

func (d *subnetworkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnetwork"
}

func (d *subnetworkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				Required: true,
			},
			"cidr": schema.StringAttribute{
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *subnetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var id types.Int32
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientReq := &pb.SubnetworkIdentificationRequest{
		Id: uint32(id.ValueInt32()),
	}

	subnetwork, err := d.client.Get(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading bx2cloud subnetwork",
			"Could not read bx2cloud subnetwork id "+strconv.FormatInt(int64(id.ValueInt32()), 10)+": "+err.Error(),
		)
		return
	}

	cidr := fmt.Sprintf("%d.%d.%d.%d/%d",
		byte(subnetwork.Address>>24),
		byte(subnetwork.Address>>16),
		byte(subnetwork.Address>>8),
		byte(subnetwork.Address),
		subnetwork.PrefixLength)

	state := subnetworkDataSourceModel{
		Id:        types.Int32Value(int32(subnetwork.Id)),
		Cidr:      types.StringValue(cidr),
		CreatedAt: types.StringValue(subnetwork.CreatedAt.AsTime().Format(time.RFC3339)),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
