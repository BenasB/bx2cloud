package terraform

import (
	"context"
	"os"

	"github.com/BenasB/bx2cloud/internal/api/pb"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type bx2cloudProviderModel struct {
	Host types.String `tfsdk:"host"`
}

type Bx2cloudClients struct {
	Network    pb.NetworkServiceClient
	Subnetwork pb.SubnetworkServiceClient
	Container  pb.ContainerServiceClient
}

var _ provider.Provider = &bx2cloudProvider{}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &bx2cloudProvider{
			version: version,
		}
	}
}

type bx2cloudProvider struct {
	version string
}

func (p *bx2cloudProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "bx2cloud"
	resp.Version = p.version
}

func (p *bx2cloudProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with bx2cloud.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "The host and port of the bx2cloud API. May also be provided via BX2CLOUD_HOST environment variable.",
				Required:    true,
			},
		},
	}
}

func (p *bx2cloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config bx2cloudProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown bx2cloud API Host",
			"The provider cannot create the bx2cloud API client as there is an unknown configuration value for the bx2cloud API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the BX2CLOUD_HOST environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	host := os.Getenv("BX2CLOUD_HOST")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing bx2cloud API Host",
			"The provider cannot create the bx2cloud API client as there is a missing or empty value for the bx2cloud API host. "+
				"Set the host value in the configuration or use the BX2CLOUD_HOST environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	conn, err := grpc.NewClient(host, opts...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create bx2cloud API Client",
			"An unexpected error occurred when creating the bx2cloud API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"bx2cloud client Error: "+err.Error(),
		)
		return
	}

	clients := &Bx2cloudClients{
		Network:    pb.NewNetworkServiceClient(conn),
		Subnetwork: pb.NewSubnetworkServiceClient(conn),
		Container:  pb.NewContainerServiceClient(conn),
	}

	resp.DataSourceData = clients
	resp.ResourceData = clients
}

func (p *bx2cloudProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewNetworkDataSource,
		NewSubnetworkDataSource,
		NewContainerDataSource,
	}
}

func (p *bx2cloudProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNetworkResource,
		NewSubnetworkResource,
		NewContainerResource,
	}
}
