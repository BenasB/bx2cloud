package terraform

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &containerDataSource{}
	_ datasource.DataSourceWithConfigure = &containerDataSource{}
)

func NewContainerDataSource() datasource.DataSource {
	return &containerDataSource{}
}

type containerDataSource struct {
	client pb.ContainerServiceClient
}

type containerDataSourceModel struct {
	Id           types.String `tfsdk:"id"`
	SubnetworkId types.String `tfsdk:"subnetwork_id"`
	Ip           types.String `tfsdk:"ip"`
	Image        types.String `tfsdk:"image"`
	Status       types.String `tfsdk:"status"`
	Entrypoint   types.List   `tfsdk:"entrypoint"`
	Cmd          types.List   `tfsdk:"cmd"`
	Env          types.Map    `tfsdk:"env"`
	StartedAt    types.String `tfsdk:"started_at"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func (d *containerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = clients.Container
}

func (d *containerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container"
}

func (d *containerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required: true,
			},
			"subnetwork_id": schema.StringAttribute{
				Description: "The subnetwork this container is attached to.",
				Computed:    true,
			},
			"ip": schema.StringAttribute{
				Description: "Specifies the container's allocated address and mask prefix length in CIDR notation. For example `10.0.8.3/24`, `192.168.10.8/25`.",
				Computed:    true,
			},
			"image": schema.StringAttribute{
				Description: "The container image name from an OCI compliant registry.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of the container: `running` or `stopped`.",
				Computed:    true,
			},
			"entrypoint": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "Overides the default executable from the original image. Corresponds to Dockerfile's `ENTRYPOINT` instruction.",
				Computed:    true,
			},
			"cmd": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "Overides the default commands from the original image. Corresponds to Dockerfile's `CMD` instruction.",
				Computed:    true,
			},
			"env": schema.MapAttribute{
				ElementType: types.StringType,
				Description: "Appends extra environment variables to the default environment variables from the original image. Corresponds to Dockerfile's `ENV` instruction.",
				Computed:    true,
			},
			"started_at": schema.StringAttribute{
				Description: "The time the container was last started at.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *containerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state containerDataSourceModel
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

	clientReq := &pb.ContainerIdentificationRequest{
		Id: uint32(id),
	}

	container, err := d.client.Get(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading container",
			"Could not read container id "+state.Id.ValueString()+": "+err.Error(),
		)
		return
	}

	cidr := fmt.Sprintf("%d.%d.%d.%d/%d",
		byte(container.Address>>24),
		byte(container.Address>>16),
		byte(container.Address>>8),
		byte(container.Address),
		container.PrefixLength)

	state.Id = types.StringValue(strconv.FormatInt(int64(container.Id), 10))
	state.SubnetworkId = types.StringValue(strconv.FormatInt(int64(container.SubnetworkId), 10))
	state.Ip = types.StringValue(cidr)
	state.Image = types.StringValue(container.Image)
	state.Status = types.StringValue(container.Status)
	state.StartedAt = types.StringValue(container.StartedAt.AsTime().Format(time.RFC3339))
	state.CreatedAt = types.StringValue(container.CreatedAt.AsTime().Format(time.RFC3339))

	state.Entrypoint, diags = types.ListValueFrom(ctx, types.StringType, container.Entrypoint)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Cmd, diags = types.ListValueFrom(ctx, types.StringType, container.Cmd)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	responseEnvMap := make(map[string]string, len(container.Env))
	for _, v := range container.Env {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			resp.Diagnostics.AddError(
				"Error parsing container creation response",
				"Could not decode environment variables into a map",
			)
			return
		}
		responseEnvMap[parts[0]] = parts[1]
	}
	state.Env, diags = types.MapValueFrom(ctx, types.StringType, responseEnvMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
