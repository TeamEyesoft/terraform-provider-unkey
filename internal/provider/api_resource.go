// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"

	"github.com/TeamEyesoft/terraform-provider-unkey/internal/provider/models"
	"github.com/TeamEyesoft/terraform-provider-unkey/internal/provider/schemas"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &apiResource{}
	_ resource.ResourceWithConfigure = &apiResource{}
)

// NewApiResource is a helper function to simplify the provider implementation.
func NewApiResource() resource.Resource {
	return &apiResource{}
}

// apiResource is the resource implementation.
type apiResource struct {
	client *unkey.Unkey
}

// Metadata returns the resource type name.
func (r *apiResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api"
}

// Schema defines the schema for the resource.
func (r *apiResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schemas.ApiSchema()
}

// Create a new resource.
func (r *apiResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan models.ApiResourceModel
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

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *apiResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state models.ApiResourceModel
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

func (r *apiResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *apiResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state models.ApiResourceModel
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
func (r *apiResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
