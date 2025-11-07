// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	unkey "github.com/unkeyed/sdks/api/go/v2"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &unkeyProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &unkeyProvider{
			version: version,
		}
	}
}

// unkeyProviderModel maps provider schema data to a Go type.
type unkeyProviderModel struct {
	RootKey types.String `tfsdk:"root_key"`
}

// unkeyProvider is the provider implementation.
type unkeyProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// Metadata returns the provider type name.
func (p *unkeyProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "unkey"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *unkeyProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Unkey.",
		Attributes: map[string]schema.Attribute{
			"root_key": schema.StringAttribute{
				Description: "Root key for Unkey API. May also be provided via UNKEY_ROOT_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *unkeyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Unkey client")

	// Retrieve provider data from configuration
	var config unkeyProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.RootKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("root_key"),
			"Unknown Root Key API",
			"The provider cannot create the Unkey API client as there is an unknown configuration value for the Unkey API root key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the UNKEY_ROOT_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	rootKey := os.Getenv("UNKEY_ROOT_KEY")

	if !config.RootKey.IsNull() {
		rootKey = config.RootKey.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if rootKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("root_key"),
			"Missing Unkey API Root Key",
			"The provider cannot create the Unkey API client as there is a missing or empty value for the Unkey API root key. "+
				"Set the root key value in the configuration or use the UNKEY_ROOT_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "unkey_root_key", rootKey)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "unkey_root_key")

	tflog.Debug(ctx, "Creating Unkey client")

	client := unkey.New(
		unkey.WithSecurity(rootKey),
	)

	// Make the Unkey client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured Unkey client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *unkeyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// Resources defines the resources implemented in the provider.
func (p *unkeyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewApiResource,
	}
}
