// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"fmt"
	"regexp"

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
	_ resource.Resource              = &permissionResource{}
	_ resource.ResourceWithConfigure = &permissionResource{}
)

// NewPermissionResource is a helper function to simplify the provider implementation.
func NewPermissionResource() resource.Resource {
	return &permissionResource{}
}

// permissionResourceModel maps the resource schema data.
type permissionResourceModel struct {
	PermissionId types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Slug         types.String `tfsdk:"slug"`
	Description  types.String `tfsdk:"description"`
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
	resp.Schema = schema.Schema{
		Description: "Manages a Permission resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the Permission resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: `Creates a permission with this human-readable name that describes its purpose.
Names must be unique within your workspace to prevent conflicts during assignment.
Use clear, semantic names that developers can easily understand when building authorization logic.
Consider using hierarchical naming conventions like 'resource.action' for better organization.

Examples: 'users.read', 'billing.write', 'analytics.view', 'admin.manage'`,
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 512),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: `Creates a URL-safe identifier for this permission that can be used in APIs and integrations.
Must start with a letter and contain only letters, numbers, periods, underscores, and hyphens.
Slugs are often used in REST endpoints, configuration files, and external integrations.
Should closely match the name but in a format suitable for technical usage.
Must be unique within your workspace to ensure reliable permission lookups.

Keep slugs concise but descriptive for better developer experience.`,
				Required: true,
				Validators: []validator.String{
					// Validate string value satisfies the regular expression for alphanumeric characters
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]*$`),
						"must match Unkey Permission ID requirements (alphanumeric, may include . _ - and must start with a letter)",
					),
					stringvalidator.LengthBetween(1, 128),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: `Provides detailed documentation of what this permission grants access to.
Include information about affected resources, allowed actions, and any important limitations.
This internal documentation helps team members understand permission scope and security implications.
Not visible to end users - designed for development teams and security audits.

Consider documenting:

- What resources can be accessed
- What operations are permitted
- Any conditions or limitations
- Related permissions that might be needed`,
				Required: false,
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(128),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Create a new resource.
func (r *permissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan permissionResourceModel
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

	permission, err := r.client.Permissions.CreatePermission(ctx, components.V2PermissionsCreatePermissionRequestBody{
		Name:        plan.Name.ValueString(),
		Slug:        plan.Slug.ValueString(),
		Description: description,
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
	var state permissionResourceModel
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

	// Overwrite items with refreshed state
	state.Name = types.StringValue(api.V2PermissionsGetPermissionResponseBody.GetData().Name)
	state.Slug = types.StringValue(api.V2PermissionsGetPermissionResponseBody.GetData().Slug)
	if api.V2PermissionsGetPermissionResponseBody.GetData().Description != nil {
		state.Description = types.StringValue(*api.V2PermissionsGetPermissionResponseBody.GetData().Description)
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

func (r *permissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *permissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state permissionResourceModel
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
