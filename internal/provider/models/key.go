package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type KeyResourceModel struct {
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
	PermanentDeletion types.Bool   `tfsdk:"permanent_deletion"`
	LastUpdated       types.String `tfsdk:"last_updated"`
}
