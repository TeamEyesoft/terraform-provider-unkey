package models

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	RatelimitAttrTypes = map[string]attr.Type{
		"name":       types.StringType,
		"limit":      types.Int64Type,
		"duration":   types.Int64Type,
		"auto_apply": types.BoolType,
	}

	RatelimitObjectType = types.ObjectType{
		AttrTypes: RatelimitAttrTypes,
	}
)

type RateLimitModel struct {
	Name      types.String `tfsdk:"name"`
	Limit     types.Int64  `tfsdk:"limit"`
	Duration  types.Int64  `tfsdk:"duration"`
	AutoApply types.Bool   `tfsdk:"auto_apply"`
}
