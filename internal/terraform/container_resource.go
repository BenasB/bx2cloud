package terraform

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &containerResource{}
	_ resource.ResourceWithConfigure   = &containerResource{}
	_ resource.ResourceWithImportState = &containerResource{}
	_ planmodifier.String              = startedAtPlanModifier{}
)

func NewContainerResource() resource.Resource {
	return &containerResource{}
}

type containerResource struct {
	client pb.ContainerServiceClient
}

type containerResourceModel struct {
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
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (r *containerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = clients.Container
}

func (r *containerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container"
}

func (r *containerResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subnetwork_id": schema.StringAttribute{
				Description: "The subnetwork this container is attached to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ip": schema.StringAttribute{
				Description: "Specifies the container's allocated address and mask prefix length in CIDR notation. For example `10.0.8.3/24`, `192.168.10.8/25`.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"image": schema.StringAttribute{
				Description: "The container image name from an OCI compliant registry.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "The status of the container: `running` or `stopped`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("running", "stopped"),
				},
			},
			"entrypoint": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "Overides the default executable from the original image. Corresponds to Dockerfile's `ENTRYPOINT` instruction.",
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"cmd": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "Overides the default commands from the original image. Corresponds to Dockerfile's `CMD` instruction.",
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"env": schema.MapAttribute{
				ElementType: types.StringType,
				Description: "Appends extra environment variables to the default environment variables from the original image. Corresponds to Dockerfile's `ENV` instruction.",
				Optional:    true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"started_at": schema.StringAttribute{
				Description: "The time the container was last started at.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					startedAtPlanModifier{},
				},
			},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *containerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan containerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	subnetworkId, err := strconv.ParseInt(plan.SubnetworkId.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("subnetwork_id"),
			"Invalid subnetwork_id Format",
			fmt.Sprintf("Could not parse subnetwork_id into an integer: %v", err),
		)
		return
	}

	if plan.Status.ValueString() == "stopped" {
		resp.Diagnostics.AddAttributeError(
			path.Root("status"),
			"Invalid status value during creation",
			"Creating a container with the initial status 'stopped' is not supported. Please create a container and then explicitly update the 'status' attribute.",
		)
		return
	}

	entrypoint := make([]string, 0)
	diags = plan.Entrypoint.ElementsAs(ctx, &entrypoint, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cmd := make([]string, 0)
	diags = plan.Cmd.ElementsAs(ctx, &cmd, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	envMap := make(map[string]string)
	diags = plan.Env.ElementsAs(ctx, &envMap, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	env := make([]string, 0, len(envMap))
	for k, v := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	clientReq := &pb.ContainerCreationRequest{
		SubnetworkId: uint32(subnetworkId),
		Image:        plan.Image.ValueString(),
		Entrypoint:   entrypoint,
		Cmd:          cmd,
		Env:          env,
	}

	container, err := r.client.Create(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating container",
			"Could not create container, unexpected error: "+err.Error(),
		)
		return
	}

	diags = r.populateFromResponse(ctx, &plan, container)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *containerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state containerResourceModel
	diags := req.State.Get(ctx, &state)
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

	container, err := r.client.Get(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading container",
			"Could not read container id "+state.Id.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = r.populateFromResponse(ctx, &state, container)
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

func (r *containerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan containerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(plan.Id.ValueString(), 10, 32)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Invalid id Format",
			fmt.Sprintf("Could not parse id into an integer: %v", err),
		)
		return
	}

	var state containerResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idReq := &pb.ContainerIdentificationRequest{
		Id: uint32(id),
	}

	var container *pb.Container
	switch {
	case state.Status.ValueString() == "stopped" && plan.Status.ValueString() == "running":
		container, err = r.client.Start(ctx, idReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating container",
				"Could not start the container, unexpected error: "+err.Error(),
			)
			return
		}
	case state.Status.ValueString() == "running" && plan.Status.ValueString() == "stopped":
		container, err = r.client.Stop(ctx, idReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating container",
				"Could not stop the container, unexpected error: "+err.Error(),
			)
			return
		}
	default:
		resp.Diagnostics.AddError(
			"Error updating container",
			"This type of container update is not supported",
		)
		return
	}

	diags = r.populateFromResponse(ctx, &plan, container)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *containerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state containerResourceModel
	diags := req.State.Get(ctx, &state)
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

	_, err = r.client.Delete(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting container",
			"Could not delete container id "+state.Id.ValueString()+": "+err.Error(),
		)
		return
	}
}

func (r *containerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type startedAtPlanModifier struct{}

func (m startedAtPlanModifier) Description(_ context.Context) string {
	return "Ensure that 'started_at' is recomputed only when the container transitions from a stopped to a running state."
}

func (m startedAtPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m startedAtPlanModifier) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	if req.State.Raw.IsNull() {
		return
	}

	if !req.PlanValue.IsUnknown() {
		return
	}

	var priorState, plannedState containerResourceModel

	diags := req.State.Get(ctx, &priorState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.Plan.Get(ctx, &plannedState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	priorStatus := priorState.Status.ValueString()
	plannedStatus := plannedState.Status.ValueString()

	if priorStatus == "stopped" && plannedStatus == "running" {
		resp.PlanValue = types.StringUnknown()
		return
	}

	resp.PlanValue = req.StateValue
}

func (r *containerResource) populateFromResponse(ctx context.Context, model *containerResourceModel, response *pb.Container) diag.Diagnostics {
	diags := make(diag.Diagnostics, 0)
	cidr := fmt.Sprintf("%d.%d.%d.%d/%d",
		byte(response.Address>>24),
		byte(response.Address>>16),
		byte(response.Address>>8),
		byte(response.Address),
		response.PrefixLength)

	model.Id = types.StringValue(strconv.FormatInt(int64(response.Id), 10))
	model.SubnetworkId = types.StringValue(strconv.FormatInt(int64(response.SubnetworkId), 10))
	model.Ip = types.StringValue(cidr)
	model.Image = types.StringValue(response.Image)
	model.Status = types.StringValue(response.Status)
	model.StartedAt = types.StringValue(response.StartedAt.AsTime().Format(time.RFC3339))
	model.CreatedAt = types.StringValue(response.CreatedAt.AsTime().Format(time.RFC3339))
	model.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	model.Entrypoint, diags = types.ListValueFrom(ctx, types.StringType, response.Entrypoint)
	if diags.HasError() {
		return diags
	}

	model.Cmd, diags = types.ListValueFrom(ctx, types.StringType, response.Cmd)
	if diags.HasError() {
		return diags
	}

	if len(response.Env) > 0 {
		responseEnvMap := make(map[string]string, len(response.Env))
		for _, v := range response.Env {
			parts := strings.SplitN(v, "=", 2)
			if len(parts) != 2 {
				diags.AddError(
					"Error parsing container creation response",
					"Could not decode environment variables into a map",
				)
				return diags
			}
			responseEnvMap[parts[0]] = parts[1]
		}

		model.Env, diags = types.MapValueFrom(ctx, types.StringType, responseEnvMap)
		if diags.HasError() {
			return diags
		}
	}

	return diags
}
