package schemas

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func IdentitySchema() schema.Schema {
	return schema.Schema{
		Description: "Manages an Identity resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: `The id of the Identity.
This is a unique identifier assigned to the Identity upon creation.`,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"external_id": schema.StringAttribute{
				MarkdownDescription: `Creates an identity using your system's unique identifier for a user, organization, or entity.
Must be stable and unique across your workspace - duplicate externalIds return CONFLICT errors.
This identifier links Unkey identities to your authentication system, database records, or tenant structure.

Avoid changing externalIds after creation as this breaks the link between your systems.
Use consistent identifier patterns across your application for easier management and debugging.
Accepts letters, numbers, underscores, dots, and hyphens for flexible identifier formats.
Essential for implementing proper multi-tenant isolation and user-specific rate limiting.`,
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
			"meta": schema.StringAttribute{
				MarkdownDescription: `Stores arbitrary JSON metadata returned during key verification for contextual information.
Eliminates additional database lookups during verification, improving performance for stateless services.
Avoid storing sensitive data here as it's returned in verification responses.

Large metadata objects increase verification latency and should stay under 10KB total size.
Use this for subscription details, feature flags, user preferences, and organization information.
Metadata is returned as-is whenever keys associated with this identity are verified.`,
				Required: false,
				Optional: true,
			},
			"ratelimits": schema.ListNestedAttribute{
				MarkdownDescription: `Defines shared rate limits that apply to all keys belonging to this identity.
Prevents abuse by users with multiple keys by enforcing consistent limits across their entire key portfolio.
Essential for implementing fair usage policies and tiered access levels in multi-tenant applications.

Rate limit counters are shared across all keys with this identity, regardless of how many keys the user creates.
During verification, specify which named limits to check for enforcement.
Identity rate limits supplement any key-specific rate limits that may also be configured.

Each named limit can have different thresholds and windows
When verifying keys, you can specify which limits you want to use and all keys attached to this identity will share the limits, regardless of which specific key is used.`,
				Required: false,
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: `The name of this rate limit. This name is used to identify which limit to check during key verification.

Best practices for limit names:

- Use descriptive, semantic names like 'api_requests', 'heavy_operations', or 'downloads'
- Be consistent with naming conventions across your application
- Create separate limits for different resource types or operation costs
- Consider using namespaced names for better organization (e.g., 'files.downloads', 'compute.training')

You will reference this exact name when verifying keys to check against this specific limit.`,
							Required: true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(3, 128),
							},
						},
						"limit": schema.Int64Attribute{
							MarkdownDescription: `The maximum number of operations allowed within the specified time window.

When this limit is reached, verification requests will fail with code=RATE_LIMITED until the window resets. The limit should reflect:

- Your infrastructure capacity and scaling limitations
- Fair usage expectations for your service
- Different tier levels for various user types
- The relative cost of the operations being limited

Higher values allow more frequent access but may impact service performance.`,
							Required: true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
						"duration": schema.Int64Attribute{
							MarkdownDescription: `The duration for each ratelimit window in milliseconds.

This controls how long the rate limit counter accumulates before resetting. Common values include:

- 1000 (1 second): For strict per-second limits on high-frequency operations
- 60000 (1 minute): For moderate API usage control
- 3600000 (1 hour): For less frequent but costly operations
- 86400000 (24 hours): For daily quotas

Shorter windows provide more frequent resets but may allow large burst usage. Longer windows provide more consistent usage patterns but take longer to reset after limit exhaustion.`,
							Required: true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1000),
							},
						},
						"auto_apply": schema.BoolAttribute{
							Description: "Whether this ratelimit should be automatically applied when verifying a key.",
							Required:    true,
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(50),
				},
			},
		},
	}
}
