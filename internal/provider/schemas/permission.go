package schemas

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func PermissionSchema() schema.Schema {
	return schema.Schema{
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
