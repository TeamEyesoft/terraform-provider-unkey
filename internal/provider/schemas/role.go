package schemas

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func RoleSchema() schema.Schema {
	return schema.Schema{
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
				MarkdownDescription: `The unique name for this role. Must be unique within your workspace and clearly indicate the role's purpose. Use descriptive names like 'admin', 'editor', or 'billing_manager'.

Examples: 'admin.billing', 'support.readonly', 'developer.api', 'manager.analytics'`,
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 512),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: `Provides comprehensive documentation of what this role encompasses and what access it grants.
Include information about the intended use case, what permissions should be assigned, and any important considerations.
This internal documentation helps team members understand role boundaries and security implications.
Not visible to end users - designed for administration teams and access control audits.

Consider documenting:

- The role's intended purpose and scope
- What types of users should receive this role
- What permissions are typically associated with it
- Any security considerations or limitations
- Related roles that might be used together`,
				Required: false,
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(2048),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}
