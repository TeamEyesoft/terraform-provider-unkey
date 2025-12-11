// Copyright (c) HashiCorp, Inc.

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &keyResource{}
	_ resource.ResourceWithConfigure = &keyResource{}

	creditsRefillAttrTypes = map[string]attr.Type{
		"interval":   types.StringType,
		"amount":     types.Int64Type,
		"refill_day": types.Int64Type,
	}

	creditsAttrTypes = map[string]attr.Type{
		"remaining": types.Int64Type,
		"refill": types.ObjectType{
			AttrTypes: creditsRefillAttrTypes,
		},
	}

	ratelimitAttrTypes = map[string]attr.Type{
		"name":       types.StringType,
		"limit":      types.Int64Type,
		"duration":   types.Int64Type,
		"auto_apply": types.BoolType,
	}

	ratelimitObjectType = types.ObjectType{
		AttrTypes: ratelimitAttrTypes,
	}
)

// NewkeyResource is a helper function to simplify the provider implementation.
func NewKeyResource() resource.Resource {
	return &keyResource{}
}

type keyCreditsRefillModel struct {
	Interval  types.String `tfsdk:"interval"`
	Amount    types.Int64  `tfsdk:"amount"`
	RefillDay types.Int64  `tfsdk:"refill_day"`
}

type keyCreditsModel struct {
	Remaining types.Int64  `tfsdk:"remaining"`
	Refill    types.Object `tfsdk:"refill"`
}

type rateLimitModel struct {
	Name      types.String `tfsdk:"name"`
	Limit     types.Int64  `tfsdk:"limit"`
	Duration  types.Int64  `tfsdk:"duration"`
	AutoApply types.Bool   `tfsdk:"auto_apply"`
}

// apiResourceModel maps the resource schema data.
type keyResourceModel struct {
	KeyId             types.String `tfsdk:"id"`
	Key               types.String `tfsdk:"key"`
	ApiId             types.String `tfsdk:"api_id"`
	Prefix            types.String `tfsdk:"prefix"`
	Name              types.String `tfsdk:"name"`
	ByteLength        types.Int64  `tfsdk:"byte_length"`
	ExternalId        types.String `tfsdk:"external_id"`
	Meta              types.String `tfsdk:"meta"`
	Roles             types.List   `tfsdk:"roles"`
	Permissions       types.List   `tfsdk:"permissions"`
	Expires           types.Int64  `tfsdk:"expires"`
	Credits           types.Object `tfsdk:"credits"`
	Ratelimits        types.List   `tfsdk:"ratelimits"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	Recoverable       types.Bool   `tfsdk:"recoverable"`
	LastUpdated       types.String `tfsdk:"last_updated"`
	PermanentDeletion types.Bool   `tfsdk:"permanent_deletion"`
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
	resp.Schema = schema.Schema{
		MarkdownDescription: `Create a new API key for user authentication and authorization.

Use this endpoint when users sign up, upgrade subscription tiers, or need additional keys. Keys are cryptographically secure and unique to the specified API namespace.

Important: The key is returned only once. Store it immediately and provide it to your user, as it cannot be retrieved later.

## Common use cases:

- Generate keys for new user registrations
- Create additional keys for different applications
- Issue keys with specific permissions or limits

## Required Permissions

Your root key needs one of:

- api.*.create_key (create keys in any API)
- api.<api_id>.create_key (create keys in specific API)`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: `The unique identifier for this key in Unkey's system.
This is NOT the actual API key, but a reference ID used for management operations like updating or deleting the key.
Store this ID in your database to reference the key later. This ID is not sensitive and can be logged or displayed in dashboards.`,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: `The full generated API key that should be securely provided to your user.
SECURITY WARNING: This is the only time you'll receive the complete key - Unkey only stores a securely hashed version. Never log or store this value in your own systems; provide it directly to your end user via secure channels. After this API call completes, this value cannot be retrieved again (unless created with recoverable=true).`,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Sensitive: true,
			},
			"api_id": schema.StringAttribute{
				MarkdownDescription: `The API namespace this key belongs to.
Keys from different APIs cannot access each other.`,
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 255),
				},
			},
			"prefix": schema.StringAttribute{
				MarkdownDescription: `Adds a visual identifier to the beginning of the generated key for easier recognition in logs and dashboards.
The prefix becomes part of the actual key string (e.g., prod_xxxxxxxxx).
Avoid using sensitive information in prefixes as they may appear in logs and error messages.`,
				Required: false,
				Optional: true,
				Validators: []validator.String{
					// Validate string value satisfies the regular expression for alphanumeric characters
					stringvalidator.LengthBetween(1, 16),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: `Sets a human-readable identifier for internal organization and dashboard display.
Never exposed to end users, only visible in management interfaces and API responses.
Avoid generic names like "API Key" when managing multiple keys for the same user or service.`,
				Required: false,
				Optional: true,
				Validators: []validator.String{
					// Validate string value satisfies the regular expression for alphanumeric characters
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"byte_length": schema.Int64Attribute{
				MarkdownDescription: `Controls the cryptographic strength of the generated key in bytes.
Higher values increase security but result in longer keys that may be more annoying to handle.
The default 16 bytes provides 2^128 possible combinations, sufficient for most applications.
Consider 32 bytes for highly sensitive APIs, but avoid values above 64 bytes unless specifically required.`,
				Required: true,
				Validators: []validator.Int64{
					int64validator.Between(16, 255),
				},
			},
			"external_id": schema.StringAttribute{
				MarkdownDescription: `Links this key to a user or entity in your system using your own identifier.
Returned during verification to identify the key owner without additional database lookups.
Essential for user-specific analytics, billing, and multi-tenant key management.
Use your primary user ID, organization ID, or tenant ID for best results.
Accepts letters, numbers, underscores, dots, and hyphens for flexible identifier formats.`,
				Required: false,
				Optional: true,
				Validators: []validator.String{
					// Validate string value satisfies the regular expression for alphanumeric characters
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"meta": schema.StringAttribute{
				MarkdownDescription: `Links this key to a user or entity in your system using your own identifier.
Returned during verification to identify the key owner without additional database lookups.
Essential for user-specific analytics, billing, and multi-tenant key management.
Use your primary user ID, organization ID, or tenant ID for best results.
Accepts letters, numbers, underscores, dots, and hyphens for flexible identifier formats.`,
				Required: false,
				Optional: true,
			},
			"roles": schema.ListAttribute{
				MarkdownDescription: `Assigns existing roles to this key for permission management through role-based access control.
Roles must already exist in your workspace before assignment.
During verification, all permissions from assigned roles are checked against requested permissions.
Roles provide a convenient way to group permissions and apply consistent access patterns across multiple keys.`,
				Required: false,
				Optional: true,
				Validators: []validator.List{
					listvalidator.SizeAtMost(100),
					listvalidator.ValueStringsAre(
						stringvalidator.LengthBetween(1, 100),
					),
				},
				ElementType: types.StringType,
			},
			"permissions": schema.ListAttribute{
				MarkdownDescription: `Grants specific permissions directly to this key without requiring role membership.
Wildcard permissions like 'documents.*' grant access to all sub-permissions including 'documents.read' and 'documents.write'.
Direct permissions supplement any permissions inherited from assigned roles.`,
				Required: false,
				Optional: true,
				Validators: []validator.List{
					listvalidator.SizeAtMost(1000),
					listvalidator.ValueStringsAre(
						stringvalidator.LengthBetween(1, 100),
					),
				},
				ElementType: types.StringType,
			},
			"expires": schema.Int64Attribute{
				MarkdownDescription: `Sets when this key automatically expires as a Unix timestamp in milliseconds.
Verification fails with code=EXPIRED immediately after this time passes.
Omitting this field creates a permanent key that never expires.

Avoid setting timestamps in the past as they immediately invalidate the key.
Keys expire based on server time, not client time, which prevents timezone-related issues.
Essential for trial periods, temporary access, and security compliance requiring key rotation.`,
				Required: false,
				Optional: true,
				Validators: []validator.Int64{
					int64validator.Between(0, 4102444800000),
				},
			},
			"credits": schema.SingleNestedAttribute{
				Description: `Controls usage-based limits through credit consumption with optional automatic refills.
Unlike rate limits which control frequency, credits control total usage with global consistency.
Essential for implementing usage-based pricing, subscription tiers, and hard usage quotas.
Omitting this field creates unlimited usage, while setting null is not allowed during creation.`,
				Required: false,
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"remaining": schema.Int64Attribute{
						Description: "Number of credits remaining (null for unlimited).",
						Required:    true,
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
						},
					},
					"refill": schema.SingleNestedAttribute{
						Description: "Configuration for automatic credit refill behavior.",
						Required:    false,
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"interval": schema.StringAttribute{
								Description: "How often credits are automatically refilled.",
								Required:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("daily", "monthly"),
								},
							},
							"amount": schema.Int64Attribute{
								Description: "Number of credits to add during each refill cycle.",
								Required:    true,
								Validators: []validator.Int64{
									int64validator.AtLeast(1),
								},
							},
							"refill_day": schema.Int64Attribute{
								MarkdownDescription: `Day of the month for monthly refills (1-31).
Only required when interval is 'monthly'.
For days beyond the month's length, refill occurs on the last day of the month.`,
								Required: false,
								Optional: true,
								Validators: []validator.Int64{
									int64validator.Between(1, 31),
								},
							},
						},
					},
				},
			},
			"ratelimits": schema.ListNestedAttribute{
				MarkdownDescription: `Defines time-based rate limits that protect against abuse by controlling request frequency.
Unlike credits which track total usage, rate limits reset automatically after each window expires.
Multiple rate limits can control different operation types with separate thresholds and windows.
Essential for preventing API abuse while maintaining good performance for legitimate usage.`,
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
			"enabled": schema.BoolAttribute{
				MarkdownDescription: `Controls whether the key is active immediately upon creation.
When set to 'false', the key exists but all verification attempts fail with 'code=DISABLED'.
Useful for pre-creating keys that will be activated later or for keys requiring manual approval.
Most keys should be created with 'enabled=true' for immediate use.`,
				Required: false,
				Optional: true,
			},
			"recoverable": schema.BoolAttribute{
				MarkdownDescription: `Controls whether the plaintext key is stored in an encrypted vault for later retrieval.
When true, allows recovering the actual key value using keys.getKey with decrypt=true.
When false, the key value cannot be retrieved after creation for maximum security.
Only enable for development keys or when key recovery is absolutely necessary.`,
				Required: false,
				Optional: true,
			},
			"permanent_deletion": schema.BoolAttribute{
				MarkdownDescription: `Controls deletion behavior between recoverable soft-deletion and irreversible permanent erasure.
Soft deletion (default) preserves key data for potential recovery through direct database operations.
Permanent deletion completely removes all traces including hash values and metadata with no recovery option.

Use permanent deletion only for regulatory compliance (GDPR), resolving hash collisions, or when reusing identical key strings.
Permanent deletion cannot be undone and may affect analytics data that references the deleted key.
Most applications should use soft deletion to maintain audit trails and prevent accidental data loss.`,
				Required: false,
				Optional: true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the API.",
				Computed:    true,
			},
		},
	}
}

// Create a new resource.
func (r *keyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan keyResourceModel
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

	if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
		var roles []string
		diags := plan.Roles.ElementsAs(ctx, &roles, false)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			request.Roles = roles
		}
	}

	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var permissions []string
		diags := plan.Permissions.ElementsAs(ctx, &permissions, false)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			request.Permissions = permissions
		}
	}

	if !plan.Credits.IsNull() && !plan.Credits.IsUnknown() {
		var credits keyCreditsModel
		diags := plan.Credits.As(ctx, &credits, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			// Map
			var refill keyCreditsRefillModel
			diags := credits.Refill.As(ctx, &refill, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
			var refillData *components.KeyCreditsRefill
			if !resp.Diagnostics.HasError() && !plan.Credits.IsNull() && !plan.Credits.IsUnknown() {
				refillData = &components.KeyCreditsRefill{
					Interval:  components.KeyCreditsRefillInterval(refill.Interval.ValueString()),
					Amount:    refill.Amount.ValueInt64(),
					RefillDay: refill.RefillDay.ValueInt64Pointer(),
				}
			}
			request.Credits = &components.KeyCreditsData{
				Remaining: credits.Remaining.ValueInt64Pointer(),
				Refill:    refillData,
			}
		}
	}

	if !plan.Meta.IsNull() {
		var metaMap map[string]any
		err := json.Unmarshal([]byte(plan.Meta.ValueString()), &metaMap)
		if err != nil {
			resp.Diagnostics.AddError("Invalid JSON in meta", err.Error())
			return
		}
		request.Meta = metaMap
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
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

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
	var state keyResourceModel
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

	// Overwrite items with refreshed state
	state.Name = types.StringValue(*key.V2KeysGetKeyResponseBody.GetData().Name)
	state.Enabled = types.BoolValue(key.V2KeysGetKeyResponseBody.GetData().Enabled)
	expires := key.V2KeysGetKeyResponseBody.GetData().Expires
	if expires != nil {
		state.Expires = types.Int64Value(*expires)
	} else {
		state.Expires = types.Int64Null()
	}
	state.Permissions, diags = types.ListValueFrom(ctx, types.StringType, key.V2KeysGetKeyResponseBody.GetData().Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Roles, diags = types.ListValueFrom(ctx, types.StringType, key.V2KeysGetKeyResponseBody.GetData().Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	metaData := key.V2KeysGetKeyResponseBody.GetData().Meta
	if metaData != nil {
		jsonBytes, _ := json.Marshal(metaData)
		state.Meta = types.StringValue(string(jsonBytes))
	} else {
		state.Meta = types.StringNull()
	}
	credits := key.V2KeysGetKeyResponseBody.GetData().Credits
	if credits != nil {
		// Build refill object
		var refillValue types.Object
		if credits.Refill != nil {
			refillModel := keyCreditsRefillModel{
				Interval:  types.StringValue(string(credits.Refill.Interval)),
				Amount:    types.Int64Value(credits.Refill.Amount),
				RefillDay: types.Int64PointerValue(credits.Refill.RefillDay),
			}

			refillValue, diags = types.ObjectValueFrom(ctx, creditsRefillAttrTypes, refillModel)
			resp.Diagnostics.Append(diags...)
		} else {
			refillValue = types.ObjectNull(creditsRefillAttrTypes)
		}

		// Build credits object
		creditsModel := keyCreditsModel{
			Remaining: types.Int64PointerValue(credits.Remaining),
			Refill:    refillValue,
		}

		state.Credits, diags = types.ObjectValueFrom(ctx, creditsAttrTypes, creditsModel)
		resp.Diagnostics.Append(diags...)
	} else {
		state.Credits = types.ObjectNull(creditsAttrTypes)
	}

	ratelimits := key.V2KeysGetKeyResponseBody.GetData().Ratelimits
	if len(ratelimits) > 0 {
		ratelimitModels := make([]rateLimitModel, len(ratelimits))

		for i, rl := range ratelimits {
			ratelimitModels[i] = rateLimitModel{
				Name:      types.StringValue(rl.Name),
				Limit:     types.Int64Value(rl.Limit),
				Duration:  types.Int64Value(rl.Duration),
				AutoApply: types.BoolValue(rl.AutoApply),
			}
		}

		state.Ratelimits, diags = types.ListValueFrom(ctx, ratelimitObjectType, ratelimitModels)
		resp.Diagnostics.Append(diags...)
	} else {
		state.Ratelimits = types.ListNull(ratelimitObjectType)
	}

	identity := key.V2KeysGetKeyResponseBody.GetData().Identity
	if identity != nil {
		state.ExternalId = types.StringValue(identity.ExternalID)
	} else {
		state.ExternalId = types.StringNull()
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *keyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get current state and plan
	var state, plan keyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keyId := state.KeyId.ValueString()

	// Build update request - only include fields that can be updated
	request := components.V2KeysUpdateKeyRequestBody{
		KeyID: keyId,
	}

	// Check what changed and update accordingly

	// Update name
	if !plan.Name.Equal(state.Name) && !plan.Name.IsNull() {
		request.Name = plan.Name.ValueStringPointer()
	}

	// Update enabled status
	if !plan.Enabled.Equal(state.Enabled) && !plan.Enabled.IsNull() {
		request.Enabled = plan.Enabled.ValueBoolPointer()
	}

	// Update expires
	if !plan.Expires.Equal(state.Expires) {
		request.Expires = plan.Expires.ValueInt64Pointer()
	}

	// Update meta
	if !plan.Meta.IsNull() {
		var metaMap map[string]any
		err := json.Unmarshal([]byte(plan.Meta.ValueString()), &metaMap)
		if err != nil {
			resp.Diagnostics.AddError("Invalid JSON in meta", err.Error())
			return
		}
		request.Meta = metaMap
	}

	// Update roles
	if !plan.Roles.Equal(state.Roles) {
		if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
			var roles []string
			diags := plan.Roles.ElementsAs(ctx, &roles, false)
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() {
				request.Roles = roles
			}
		} else {
			request.Roles = []string{}
		}
	}

	// Update permissions
	if !plan.Permissions.Equal(state.Permissions) {
		if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
			var permissions []string
			diags := plan.Permissions.ElementsAs(ctx, &permissions, false)
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() {
				request.Permissions = permissions
			}
		} else {
			request.Permissions = []string{}
		}
	}

	// Update credits
	if !plan.Credits.Equal(state.Credits) {
		if !plan.Credits.IsNull() && !plan.Credits.IsUnknown() {
			var credits keyCreditsModel
			diags := plan.Credits.As(ctx, &credits, basetypes.ObjectAsOptions{})
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() {
				var refillData *components.UpdateKeyCreditsRefill

				if !credits.Refill.IsNull() && !credits.Refill.IsUnknown() {
					var refill keyCreditsRefillModel
					diags := credits.Refill.As(ctx, &refill, basetypes.ObjectAsOptions{})
					resp.Diagnostics.Append(diags...)
					if !resp.Diagnostics.HasError() {
						refillData = &components.UpdateKeyCreditsRefill{
							Interval:  components.UpdateKeyCreditsRefillInterval(refill.Interval.ValueString()),
							Amount:    refill.Amount.ValueInt64(),
							RefillDay: refill.RefillDay.ValueInt64Pointer(),
						}
					}
				}

				request.Credits = &components.UpdateKeyCreditsData{
					Remaining: credits.Remaining.ValueInt64Pointer(),
					Refill:    refillData,
				}
			}
		} else {
			request.Credits = nil
		}
	}

	// Update ratelimits
	if !plan.Ratelimits.Equal(state.Ratelimits) {
		if !plan.Ratelimits.IsNull() && !plan.Ratelimits.IsUnknown() {
			var ratelimits []rateLimitModel
			diags := plan.Ratelimits.ElementsAs(ctx, &ratelimits, false)
			resp.Diagnostics.Append(diags...)
			if !resp.Diagnostics.HasError() {
				request.Ratelimits = make([]components.RatelimitRequest, len(ratelimits))
				for i, rl := range ratelimits {
					autoApply := rl.AutoApply.ValueBool()
					request.Ratelimits[i] = components.RatelimitRequest{
						Name:      rl.Name.ValueString(),
						Limit:     rl.Limit.ValueInt64(),
						Duration:  rl.Duration.ValueInt64(),
						AutoApply: &autoApply,
					}
				}
			}
		} else {
			request.Ratelimits = []components.RatelimitRequest{}
		}
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

	// Update state with API response
	plan.Name = types.StringPointerValue(key.V2KeysGetKeyResponseBody.GetData().Name)
	plan.Enabled = types.BoolValue(key.V2KeysGetKeyResponseBody.GetData().Enabled)

	expires := key.V2KeysGetKeyResponseBody.GetData().Expires
	if expires != nil {
		plan.Expires = types.Int64Value(*expires)
	} else {
		plan.Expires = types.Int64Null()
	}

	plan.Permissions, diags = types.ListValueFrom(ctx, types.StringType, key.V2KeysGetKeyResponseBody.GetData().Permissions)
	resp.Diagnostics.Append(diags...)

	plan.Roles, diags = types.ListValueFrom(ctx, types.StringType, key.V2KeysGetKeyResponseBody.GetData().Roles)
	resp.Diagnostics.Append(diags...)

	metaData := key.V2KeysGetKeyResponseBody.GetData().Meta
	if metaData != nil {
		jsonBytes, _ := json.Marshal(metaData)
		plan.Meta = types.StringValue(string(jsonBytes))
	} else {
		plan.Meta = types.StringNull()
	}

	// Convert Credits
	credits := key.V2KeysGetKeyResponseBody.GetData().Credits
	if credits != nil {
		var refillValue types.Object
		if credits.Refill != nil {
			refillModel := keyCreditsRefillModel{
				Interval:  types.StringValue(string(credits.Refill.Interval)),
				Amount:    types.Int64Value(credits.Refill.Amount),
				RefillDay: types.Int64PointerValue(credits.Refill.RefillDay),
			}
			refillValue, diags = types.ObjectValueFrom(ctx, creditsRefillAttrTypes, refillModel)
			resp.Diagnostics.Append(diags...)
		} else {
			refillValue = types.ObjectNull(creditsRefillAttrTypes)
		}

		creditsModel := keyCreditsModel{
			Remaining: types.Int64PointerValue(credits.Remaining),
			Refill:    refillValue,
		}

		plan.Credits, diags = types.ObjectValueFrom(ctx, creditsAttrTypes, creditsModel)
		resp.Diagnostics.Append(diags...)
	} else {
		plan.Credits = types.ObjectNull(creditsAttrTypes)
	}

	// Convert Ratelimits
	ratelimits := key.V2KeysGetKeyResponseBody.GetData().Ratelimits
	if len(ratelimits) > 0 {
		ratelimitModels := make([]rateLimitModel, len(ratelimits))
		for i, rl := range ratelimits {
			ratelimitModels[i] = rateLimitModel{
				Name:      types.StringValue(rl.Name),
				Limit:     types.Int64Value(rl.Limit),
				Duration:  types.Int64Value(rl.Duration),
				AutoApply: types.BoolValue(rl.AutoApply),
			}
		}
		plan.Ratelimits, diags = types.ListValueFrom(ctx, ratelimitObjectType, ratelimitModels)
		resp.Diagnostics.Append(diags...)
	} else {
		plan.Ratelimits = types.ListNull(ratelimitObjectType)
	}

	// Update last_updated timestamp
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *keyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state keyResourceModel
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
