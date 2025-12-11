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
	_ resource.Resource              = &permissionResource{}
	_ resource.ResourceWithConfigure = &permissionResource{}
)

// NewPermissionResource is a helper function to simplify the provider implementation.
func NewPermissionResource() resource.Resource {
	return &permissionResource{}
}

// permissionResource is the resource implementation.
type permissionResource struct {
	client *unkey.Unkey
}

// Metadata returns the resource type name.
func (r *permissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

// Schema defines the schema for the resource.
func (r *permissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schemas.PermissionSchema()
}

// Create a new resource.
func (r *permissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan models.PermissionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	permission, err := r.client.Permissions.CreatePermission(ctx, components.V2PermissionsCreatePermissionRequestBody{
		Name:        plan.Name.ValueString(),
		Slug:        plan.Slug.ValueString(),
		Description: plan.Description.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Permission",
			"Could not create Permission, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.PermissionId = types.StringValue(permission.V2PermissionsCreatePermissionResponseBody.GetData().PermissionID)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *permissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state models.PermissionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed API from Unkey
	api, err := r.client.Permissions.GetPermission(ctx, components.V2PermissionsGetPermissionRequestBody{
		Permission: state.PermissionId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Unkey Permission",
			"Could not read Unkey Permission ID "+state.PermissionId.ValueString()+": "+err.Error(),
		)
		return
	}

	data := api.V2PermissionsGetPermissionResponseBody.GetData()

	// Overwrite items with refreshed state
	state.Name = types.StringValue(data.Name)
	state.Slug = types.StringValue(data.Slug)
	state.Description = types.StringPointerValue(data.Description)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *permissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *permissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state models.PermissionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing Permission
	_, err := r.client.Permissions.DeletePermission(ctx, components.V2PermissionsDeletePermissionRequestBody{
		Permission: state.PermissionId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Unkey Permission",
			"Could not delete Permission, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *permissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
