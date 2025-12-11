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
	_ resource.Resource              = &keyResource{}
	_ resource.ResourceWithConfigure = &keyResource{}
)

// NewkeyResource is a helper function to simplify the provider implementation.
func NewKeyResource() resource.Resource {
	return &keyResource{}
}

// keyResource is the resource implementation.
type keyResource struct {
	client *unkey.Unkey
}

// Metadata returns the resource type name.
func (r *keyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key"
}

// Schema defines the schema for the resource.
func (r *keyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schemas.KeySchema()
}

// Create a new resource.
func (r *keyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan models.KeyResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := components.V2KeysCreateKeyRequestBody{
		APIID:       plan.ApiId.ValueString(),
		Prefix:      plan.Prefix.ValueStringPointer(),
		Name:        plan.Name.ValueStringPointer(),
		ByteLength:  plan.ByteLength.ValueInt64Pointer(),
		ExternalID:  plan.ExternalId.ValueStringPointer(),
		Expires:     plan.Expires.ValueInt64Pointer(),
		Enabled:     plan.Enabled.ValueBoolPointer(),
		Recoverable: plan.Recoverable.ValueBoolPointer(),
	}

	request.Roles, diags = conversions.StringListToSlice(ctx, plan.Roles)
	resp.Diagnostics.Append(diags...)

	request.Permissions, diags = conversions.StringListToSlice(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)

	request.Meta, diags = conversions.StringToMap(ctx, plan.Meta)
	resp.Diagnostics.Append(diags...)

	request.Credits, diags = conversions.CreditsToAPI(ctx, plan.Credits)
	resp.Diagnostics.Append(diags...)

	request.Ratelimits, diags = conversions.RatelimitsToAPI(ctx, plan.Ratelimits)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Create new Key
	key, err := r.client.Keys.CreateKey(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating API",
			"Could not create API, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.KeyId = types.StringValue(key.V2KeysCreateKeyResponseBody.Data.KeyID)
	plan.Key = types.StringValue(key.V2KeysCreateKeyResponseBody.Data.Key)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *keyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state models.KeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed API from Unkey
	key, err := r.client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{
		KeyID: state.KeyId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Unkey Key",
			"Could not read Unkey Key ID "+state.KeyId.ValueString()+": "+err.Error(),
		)
		return
	}

	data := key.V2KeysGetKeyResponseBody.GetData()

	// Overwrite items with refreshed state
	state.Name = types.StringPointerValue(data.Name)
	state.Enabled = types.BoolValue(data.Enabled)
	state.Expires = types.Int64PointerValue(data.Expires)

	state.Permissions, diags = conversions.SliceToStringList(ctx, data.Permissions)
	resp.Diagnostics.Append(diags...)

	state.Roles, diags = conversions.SliceToStringList(ctx, data.Roles)
	resp.Diagnostics.Append(diags...)

	state.Meta, diags = conversions.MapToString(ctx, data.Meta)
	resp.Diagnostics.Append(diags...)

	state.Credits, diags = conversions.CreditsFromAPI(ctx, data.Credits)
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

func (r *keyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get current state and plan
	var state, plan models.KeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keyId := state.KeyId.ValueString()

	// Build update request - only include fields that can be updated
	request := components.V2KeysUpdateKeyRequestBody{
		KeyID:      keyId,
		Name:       plan.Name.ValueStringPointer(),
		ExternalID: plan.ExternalId.ValueStringPointer(),
		Expires:    plan.Expires.ValueInt64Pointer(),
		Enabled:    plan.Enabled.ValueBoolPointer(),
	}

	request.Roles, diags = conversions.StringListToSlice(ctx, plan.Roles)
	resp.Diagnostics.Append(diags...)

	request.Permissions, diags = conversions.StringListToSlice(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)

	request.Meta, diags = conversions.StringToMap(ctx, plan.Meta)
	resp.Diagnostics.Append(diags...)

	request.Credits, diags = conversions.CreditsToUpdateAPI(ctx, plan.Credits)
	resp.Diagnostics.Append(diags...)

	request.Ratelimits, diags = conversions.RatelimitsToAPI(ctx, plan.Ratelimits)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Make the API call
	_, err := r.client.Keys.UpdateKey(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating key",
			"Could not update key "+keyId+": "+err.Error(),
		)
		return
	}

	// Read back the updated key to get the current state
	key, err := r.client.Keys.GetKey(ctx, components.V2KeysGetKeyRequestBody{
		KeyID: keyId,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading updated key",
			"Could not read key after update "+keyId+": "+err.Error(),
		)
		return
	}

	data := key.V2KeysGetKeyResponseBody.GetData()

	// Update state with API response
	plan.Name = types.StringPointerValue(data.Name)
	plan.Enabled = types.BoolValue(data.Enabled)
	plan.Expires = types.Int64PointerValue(data.Expires)

	state.Permissions, diags = conversions.SliceToStringList(ctx, data.Permissions)
	resp.Diagnostics.Append(diags...)

	state.Roles, diags = conversions.SliceToStringList(ctx, data.Roles)
	resp.Diagnostics.Append(diags...)

	state.Meta, diags = conversions.MapToString(ctx, data.Meta)
	resp.Diagnostics.Append(diags...)

	state.Credits, diags = conversions.CreditsFromAPI(ctx, data.Credits)
	resp.Diagnostics.Append(diags...)

	state.Ratelimits, diags = conversions.RatelimitsFromAPI(ctx, data.Ratelimits)
	resp.Diagnostics.Append(diags...)

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *keyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state models.KeyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	permanentDeletion := state.PermanentDeletion.ValueBool()

	// Delete existing API
	_, err := r.client.Keys.DeleteKey(ctx, components.V2KeysDeleteKeyRequestBody{
		KeyID:     state.KeyId.ValueString(),
		Permanent: &permanentDeletion,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Unkey Key",
			"Could not delete Key, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *keyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
