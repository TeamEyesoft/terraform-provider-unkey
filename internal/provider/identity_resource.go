// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

// TODO: Finish

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &identityResource{}
	_ resource.ResourceWithConfigure = &identityResource{}
)

// NewIdentityResource is a helper function to simplify the provider implementation.
func NewIdentityResource() resource.Resource {
	return &identityResource{}
}

// apiResourceModel maps the resource schema data.
type identityResourceModel struct {
	ApiId       types.String `tfsdk:"api_id"`
	Name        types.String `tfsdk:"name"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// identityResource is the resource implementation.
type identityResource struct {
	client *unkey.Unkey
}

// Metadata returns the resource type name.
func (r *identityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity"
}

// Schema defines the schema for the resource.
func (r *identityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an API resource.",
		Attributes: map[string]schema.Attribute{
			"api_id": schema.StringAttribute{
				Description: "Unique identifier of the API resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Unique identifier of the API resource.",
				Required:    true,
				Validators: []validator.String{
					// Validate string value satisfies the regular expression for alphanumeric characters
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]*$`),
						"must match Unkey API ID requirements (alphanumeric, may include . _ - and must start with a letter)",
					),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the API.",
				Computed:    true,
			},
		},
	}
}

// Create a new resource.
func (r *identityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan identityResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new API
	api, err := r.client.Apis.CreateAPI(ctx, components.V2ApisCreateAPIRequestBody{
		Name: plan.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating API",
			"Could not create API, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.ApiId = types.StringValue(api.V2ApisCreateAPIResponseBody.GetData().APIID)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *identityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state identityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed API from Unkey
	api, err := r.client.Apis.GetAPI(ctx, components.V2ApisGetAPIRequestBody{
		APIID: state.ApiId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Unkey API",
			"Could not read Unkey API ID "+state.ApiId.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.Name = types.StringValue(api.V2ApisGetAPIResponseBody.GetData().Name)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *identityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	return
}

func (r *identityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state identityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing API
	_, err := r.client.Apis.DeleteAPI(ctx, components.V2ApisDeleteAPIRequestBody{
		APIID: state.ApiId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Unkey API",
			"Could not delete API, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *identityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unkey.Unkey)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *unkey.Unkey, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}
