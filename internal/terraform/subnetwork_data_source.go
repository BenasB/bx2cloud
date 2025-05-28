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
	Id        types.String `tfsdk:"id"`
	Cidr      types.String `tfsdk:"cidr"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (d *subnetworkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*Bx2cloudClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Bx2cloudClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = clients.Subnetwork
}

func (d *subnetworkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnetwork"
}

func (d *subnetworkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required: true,
			},
			"cidr": schema.StringAttribute{
				Description: "Specifies the subnetwork's address and mask prefix length in CIDR notation. for example 10.0.8.0/24, 192.168.10.8/30.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *subnetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state subnetworkDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(state.Id.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Invalid id Format",
			fmt.Sprintf("Could not parse id into an integer: %v", err),
		)
		return
	}

	clientReq := &pb.SubnetworkIdentificationRequest{
		Id: uint32(id),
	}

	subnetwork, err := d.client.Get(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading bx2cloud subnetwork",
			"Could not read bx2cloud subnetwork id "+state.Id.ValueString()+": "+err.Error(),
		)
		return
	}

	cidr := fmt.Sprintf("%d.%d.%d.%d/%d",
		byte(subnetwork.Address>>24),
		byte(subnetwork.Address>>16),
		byte(subnetwork.Address>>8),
		byte(subnetwork.Address),
		subnetwork.PrefixLength)

	state.Id = types.StringValue(strconv.FormatInt(int64(subnetwork.Id), 10))
	state.Cidr = types.StringValue(cidr)
	state.CreatedAt = types.StringValue(subnetwork.CreatedAt.AsTime().Format(time.RFC3339))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
