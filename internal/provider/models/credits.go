package models

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Attribute type definitions
var (
	CreditsRefillAttrTypes = map[string]attr.Type{
		"interval":   types.StringType,
		"amount":     types.Int64Type,
		"refill_day": types.Int64Type,
	}

	CreditsAttrTypes = map[string]attr.Type{
		"remaining": types.Int64Type,
		"refill": types.ObjectType{
			AttrTypes: CreditsRefillAttrTypes,
		},
	}
)

// Models
type KeyCreditsRefillModel struct {
	Interval  types.String `tfsdk:"interval"`
	Amount    types.Int64  `tfsdk:"amount"`
	RefillDay types.Int64  `tfsdk:"refill_day"`
}

type KeyCreditsModel struct {
	Remaining types.Int64  `tfsdk:"remaining"`
	Refill    types.Object `tfsdk:"refill"`
}
