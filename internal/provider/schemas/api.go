package schemas

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func ApiSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an API resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: `The unique identifier assigned to the newly created API.
Use this ID for all subsequent operations including key creation, verification, and API management.
Always begins with 'api_' followed by a unique alphanumeric sequence.

Store this ID securely as it's required when:

- Creating API keys within this namespace
- Verifying keys associated with this API
- Managing API settings and metadata
- Listing keys belonging to this API

This identifier is permanent and cannot be changed after creation.`,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: `Unique identifier for this API namespace within your workspace.
Use descriptive names like 'payment-service-prod' or 'user-api-dev' to clearly identify purpose and environment.`,
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 255),
					// Validate string value satisfies the regular expression for alphanumeric characters
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]*$`),
						"must match Unkey API ID requirements (alphanumeric, may include . _ - and must start with a letter)",
					),
				},
			},
		},
	}
}
