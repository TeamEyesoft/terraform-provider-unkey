package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PermissionResourceModel struct {
	PermissionId types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Slug         types.String `tfsdk:"slug"`
	Description  types.String `tfsdk:"description"`
}
