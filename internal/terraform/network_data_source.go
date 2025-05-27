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
	_ datasource.DataSource              = &networkDataSource{}
	_ datasource.DataSourceWithConfigure = &networkDataSource{}
)

func NewNetworkDataSource() datasource.DataSource {
	return &networkDataSource{}
}

type networkDataSource struct {
	client pb.NetworkServiceClient
}

type networkDataSourceModel struct {
	Id             types.Int32  `tfsdk:"id"`
	InternetAccess types.Bool   `tfsdk:"internet_access"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

func (d *networkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = clients.network
}

func (d *networkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (d *networkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				Required: true,
			},
			"internet_access": schema.BoolAttribute{
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *networkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var id types.Int32
	diags := req.Config.GetAttribute(ctx, path.Root("id"), &id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientReq := &pb.NetworkIdentificationRequest{
		Id: uint32(id.ValueInt32()),
	}

	network, err := d.client.Get(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading bx2cloud network",
			"Could not read bx2cloud network id "+strconv.FormatInt(int64(id.ValueInt32()), 10)+": "+err.Error(),
		)
		return
	}

	state := networkDataSourceModel{
		Id:             types.Int32Value(int32(network.Id)),
		InternetAccess: types.BoolValue(network.InternetAccess),
		CreatedAt:      types.StringValue(network.CreatedAt.AsTime().Format(time.RFC3339)),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
