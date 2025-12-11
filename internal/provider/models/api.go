package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type ApiResourceModel struct {
	ApiId types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
}
