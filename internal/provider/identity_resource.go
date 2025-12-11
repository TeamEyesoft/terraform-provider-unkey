// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"

	"github.com/TeamEyesoft/terraform-provider-unkey/internal/provider/conversions"
	"github.com/TeamEyesoft/terraform-provider-unkey/internal/provider/models"
	"github.com/TeamEyesoft/terraform-provider-unkey/internal/provider/schemas"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &identityResource{}
	_ resource.ResourceWithConfigure = &identityResource{}
)

// NewIdentityResource is a helper function to simplify the provider implementation.
func NewIdentityResource() resource.Resource {
	return &identityResource{}
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
	resp.Schema = schemas.IdentitySchema()
}

// Create a new resource.
func (r *identityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan models.IdentityResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := components.V2IdentitiesCreateIdentityRequestBody{
		ExternalID: plan.ExternalId.ValueString(),
	}

	request.Meta, diags = conversions.StringToMap(ctx, plan.Meta)
	resp.Diagnostics.Append(diags...)

	request.Ratelimits, diags = conversions.RatelimitsToAPI(ctx, plan.Ratelimits)
	resp.Diagnostics.Append(diags...)

	// Create new Identity
	identity, err := r.client.Identities.CreateIdentity(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating API",
			"Could not create API, unexpected error: "+err.Error(),
		)
		return
	}

	data := identity.V2IdentitiesCreateIdentityResponseBody.GetData()

	// Map response body to schema and populate Computed attribute values
	plan.IdentityId = types.StringValue(data.IdentityID)

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
	var state models.IdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed Identity from Unkey
	identity, err := r.client.Identities.GetIdentity(ctx, components.V2IdentitiesGetIdentityRequestBody{
		Identity: state.IdentityId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Unkey Identity",
			"Could not read Unkey Identity ID "+state.IdentityId.ValueString()+": "+err.Error(),
		)
		return
	}

	data := identity.V2IdentitiesGetIdentityResponseBody.GetData()

	// Overwrite items with refreshed state
	state.ExternalId = types.StringValue(data.ExternalID)

	state.Meta, diags = conversions.MapToString(ctx, data.Meta)
	resp.Diagnostics.Append(diags...)

	state.Ratelimits, diags = conversions.RatelimitsFromAPI(ctx, data.Ratelimits)
	resp.Diagnostics.Append(diags...)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *identityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get current state and plan
	var state, plan models.IdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	identityId := state.IdentityId.ValueString()

	// Build update request - only include fields that can be updated
	request := components.V2IdentitiesUpdateIdentityRequestBody{
		Identity: identityId,
	}

	request.Meta, diags = conversions.StringToMap(ctx, plan.Meta)
	resp.Diagnostics.Append(diags...)

	request.Ratelimits, diags = conversions.RatelimitsToAPI(ctx, plan.Ratelimits)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Make the API call
	_, err := r.client.Identities.UpdateIdentity(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating identity",
			"Could not update identity "+identityId+": "+err.Error(),
		)
		return
	}

	// Read back the updated identity to get the current state
	identity, err := r.client.Identities.GetIdentity(ctx, components.V2IdentitiesGetIdentityRequestBody{
		Identity: identityId,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading updated identity",
			"Could not read identity after update "+identityId+": "+err.Error(),
		)
		return
	}

	data := identity.V2IdentitiesGetIdentityResponseBody.GetData()

	// Update state with API response
	plan.ExternalId = types.StringValue(data.ExternalID)

	state.Meta, diags = conversions.MapToString(ctx, data.Meta)
	resp.Diagnostics.Append(diags...)

	state.Ratelimits, diags = conversions.RatelimitsFromAPI(ctx, data.Ratelimits)
	resp.Diagnostics.Append(diags...)

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *identityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state models.IdentityResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing API
	_, err := r.client.Identities.DeleteIdentity(ctx, components.V2IdentitiesDeleteIdentityRequestBody{
		Identity: state.IdentityId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Unkey Identity",
			"Could not delete Identity, unexpected error: "+err.Error(),
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
