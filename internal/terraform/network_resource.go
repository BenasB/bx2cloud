package terraform

import (
	"context"
	"fmt"
	"strconv"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &networkResource{}
	_ resource.ResourceWithConfigure   = &networkResource{}
	_ resource.ResourceWithImportState = &networkResource{}
)

func NewNetworkResource() resource.Resource {
	return &networkResource{}
}

type networkResource struct {
	client pb.NetworkServiceClient
}

type networkResourceModel struct {
	Id             types.String `tfsdk:"id"`
	InternetAccess types.Bool   `tfsdk:"internet_access"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *networkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = clients.Network
}

func (r *networkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *networkResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"internet_access": schema.BoolAttribute{
				Description: "Whether the network allows devices on it to access the internet.",
				Required:    true,
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

func (r *networkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientReq := &pb.NetworkCreationRequest{
		InternetAccess: plan.InternetAccess.ValueBool(),
	}

	network, err := r.client.Create(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating network",
			"Could not create network, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(strconv.FormatInt(int64(network.Id), 10))
	plan.InternetAccess = types.BoolValue(network.InternetAccess)
	plan.CreatedAt = types.StringValue(network.CreatedAt.AsTime().Format(time.RFC3339))
	plan.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkResourceModel
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

	clientReq := &pb.NetworkIdentificationRequest{
		Id: uint32(id),
	}

	network, err := r.client.Get(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading network",
			"Could not read network id "+state.Id.ValueString()+": "+err.Error(),
		)
		return
	}

	state.Id = types.StringValue(strconv.FormatInt(int64(network.Id), 10))
	state.InternetAccess = types.BoolValue(network.InternetAccess)
	state.CreatedAt = types.StringValue(network.CreatedAt.AsTime().Format(time.RFC3339))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan networkResourceModel
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

	clientReq := &pb.NetworkUpdateRequest{
		Identification: &pb.NetworkIdentificationRequest{
			Id: uint32(id),
		},
		Update: &pb.NetworkCreationRequest{
			InternetAccess: plan.InternetAccess.ValueBool(),
		},
	}

	network, err := r.client.Update(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating network",
			"Could not update network, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(strconv.FormatInt(int64(network.Id), 10))
	plan.InternetAccess = types.BoolValue(network.InternetAccess)
	plan.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkResourceModel
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

	clientReq := &pb.NetworkIdentificationRequest{
		Id: uint32(id),
	}

	_, err = r.client.Delete(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting network",
			"Could not delete network id "+state.Id.ValueString()+": "+err.Error(),
		)
		return
	}
}

func (r *networkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
