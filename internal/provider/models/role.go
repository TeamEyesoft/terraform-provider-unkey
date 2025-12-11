package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RoleResourceModel struct {
	RoleId      types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}
