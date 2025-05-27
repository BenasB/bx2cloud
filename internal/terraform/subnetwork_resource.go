package terraform

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	pb "github.com/BenasB/bx2cloud/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &subnetworkResource{}
	_ resource.ResourceWithConfigure = &subnetworkResource{}
)

func NewSubnetworkResource() resource.Resource {
	return &subnetworkResource{}
}

type subnetworkResource struct {
	client pb.SubnetworkServiceClient
}

type subnetworkResourceModel struct {
	Id        types.Int32  `tfsdk:"id"`
	Cidr      types.String `tfsdk:"cidr"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *subnetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = clients.subnetwork
}

func (r *subnetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnetwork"
}

func (r *subnetworkResource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int32Attribute{
				Computed: true,
			},
			"cidr": schema.StringAttribute{
				Required: true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *subnetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan subnetworkResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, ipNet, err := net.ParseCIDR(plan.Cidr.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("cidr"),
			"Invalid CIDR Format",
			fmt.Sprintf("Could not parse CIDR: %v. Expected format is <address>/<prefix> (e.g., 10.0.0.0/16)", err),
		)
		return
	}

	ip := ipNet.IP.To4()
	if ip == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("cidr"),
			"Invalid CIDR Format",
			"Could not convert the ip to an IPv4 ip",
		)
		return
	}
	address := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
	prefixLength, _ := ipNet.Mask.Size()

	clientReq := &pb.SubnetworkCreationRequest{
		Address:      address,
		PrefixLength: uint32(prefixLength),
	}

	subnetwork, err := r.client.Create(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating subnetwork",
			"Could not create subnetwork, unexpected error: "+err.Error(),
		)
		return
	}

	cidr := fmt.Sprintf("%d.%d.%d.%d/%d",
		byte(subnetwork.Address>>24),
		byte(subnetwork.Address>>16),
		byte(subnetwork.Address>>8),
		byte(subnetwork.Address),
		subnetwork.PrefixLength)

	plan.Id = types.Int32Value(int32(subnetwork.Id))
	plan.Cidr = types.StringValue(cidr)
	plan.CreatedAt = types.StringValue(subnetwork.CreatedAt.AsTime().Format(time.RFC3339))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *subnetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state subnetworkResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientReq := &pb.SubnetworkIdentificationRequest{
		Id: uint32(state.Id.ValueInt32()),
	}

	subnetwork, err := r.client.Get(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading bx2cloud subnetwork",
			"Could not read bx2cloud subnetwork id "+strconv.FormatInt(int64(state.Id.ValueInt32()), 10)+": "+err.Error(),
		)
		return
	}

	cidr := fmt.Sprintf("%d.%d.%d.%d/%d",
		byte(subnetwork.Address>>24),
		byte(subnetwork.Address>>16),
		byte(subnetwork.Address>>8),
		byte(subnetwork.Address),
		subnetwork.PrefixLength)

	state.Id = types.Int32Value(int32(subnetwork.Id))
	state.Cidr = types.StringValue(cidr)
	state.CreatedAt = types.StringValue(subnetwork.CreatedAt.AsTime().Format(time.RFC3339))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *subnetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	panic("unimplemented")
}

func (r *subnetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state subnetworkResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientReq := &pb.SubnetworkIdentificationRequest{
		Id: uint32(state.Id.ValueInt32()),
	}

	_, err := r.client.Delete(ctx, clientReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting bx2cloud subnetwork",
			"Could not delete bx2cloud subnetwork id "+strconv.FormatInt(int64(state.Id.ValueInt32()), 10)+": "+err.Error(),
		)
		return
	}
}
