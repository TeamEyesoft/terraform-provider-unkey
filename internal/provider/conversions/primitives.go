package conversions

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func StringListToSlice(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	var result []string
	diags.Append(list.ElementsAs(ctx, &result, false)...)
	return result, diags
}

func SliceToStringList(ctx context.Context, slice []string) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(slice) == 0 {
		return types.ListNull(types.StringType), diags
	}

	list, d := types.ListValueFrom(ctx, types.StringType, slice)
	diags.Append(d...)
	return list, diags
}

func StringToMap(ctx context.Context, str types.String) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	if str.IsNull() || str.IsUnknown() {
		return nil, diags
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(str.ValueString()), &result); err != nil {
		diags.AddError("Error unmarshaling string to map", err.Error())
		return nil, diags
	}

	return result, diags
}

func MapToString(ctx context.Context, m map[string]any) (types.String, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(m) == 0 {
		return types.StringNull(), diags
	}

	jsonBytes, err := json.Marshal(m)
	if err != nil {
		diags.AddError("Error marshaling map to string", err.Error())
		return types.StringNull(), diags
	}

	return types.StringValue(string(jsonBytes)), diags
}
