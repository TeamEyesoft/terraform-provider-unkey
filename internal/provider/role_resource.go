// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"

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

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &roleResource{}
	_ resource.ResourceWithConfigure = &roleResource{}
)

// NewRoleResource is a helper function to simplify the provider implementation.
func NewRoleResource() resource.Resource {
	return &roleResource{}
}

// roleResourceModel maps the resource schema data.
type roleResourceModel struct {
	RoleId      types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

// roleResource is the resource implementation.
type roleResource struct {
	client *unkey.Unkey
}

// Metadata returns the resource type name.
func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema defines the schema for the resource.
func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Role resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the Role resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: `The unique name for this role. Must be unique within your workspace and clearly indicate the role's purpose. Use descriptive names like 'admin', 'editor', or 'billing_manager'.

Examples: 'admin.billing', 'support.readonly', 'developer.api', 'manager.analytics'`,
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 512),
				},
			},
			"description": schema.StringAttribute{
				Description: `Provides comprehensive documentation of what this role encompasses and what access it grants.
Include information about the intended use case, what permissions should be assigned, and any important considerations.
This internal documentation helps team members understand role boundaries and security implications.
Not visible to end users - designed for administration teams and access control audits.

Consider documenting:

- The role's intended purpose and scope
- What types of users should receive this role
- What permissions are typically associated with it
- Any security considerations or limitations
- Related roles that might be used together
`,
				Required: false,
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(2048),
				},
			},
		},
	}
}

// Create a new resource.
func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan roleResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new Permission
	var description *string
	if !plan.Description.IsNull() {
		desc := plan.Description.ValueString()
		description = &desc
	}

	role, err := r.client.Permissions.CreateRole(ctx, components.V2PermissionsCreateRoleRequestBody{
		Name:        plan.Name.ValueString(),
		Description: description,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Role",
			"Could not create Role, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.RoleId = types.StringValue(role.V2PermissionsCreateRoleResponseBody.GetData().RoleID)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state roleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed API from Unkey
	api, err := r.client.Permissions.GetRole(ctx, components.V2PermissionsGetRoleRequestBody{
		Role: state.RoleId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Unkey Role",
			"Could not read Unkey Role ID "+state.RoleId.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.Name = types.StringValue(api.V2PermissionsGetRoleResponseBody.GetData().Name)
	if api.V2PermissionsGetRoleResponseBody.GetData().Description != nil {
		state.Description = types.StringValue(*api.V2PermissionsGetRoleResponseBody.GetData().Description)
	} else {
		state.Description = types.StringNull()
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	return
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state roleResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing Role
	_, err := r.client.Permissions.DeleteRole(ctx, components.V2PermissionsDeleteRoleRequestBody{
		Role: state.RoleId.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Unkey Role",
			"Could not delete Role, unexpected error: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
