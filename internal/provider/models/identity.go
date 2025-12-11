package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type IdentityResourceModel struct {
	IdentityId types.String `tfsdk:"id"`
	ExternalId types.String `tfsdk:"external_id"`
	Meta       types.String `tfsdk:"meta"`
	Ratelimits types.List   `tfsdk:"ratelimits"`
}
