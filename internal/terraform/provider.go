// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
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
	resp.Schema = schema.Schema{}
}

func (p *bx2cloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {

}

func (p *bx2cloudProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}

func (p *bx2cloudProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
}
