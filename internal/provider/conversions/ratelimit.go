package conversions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/unkeyed/sdks/api/go/v2/models/components"

	"github.com/TeamEyesoft/terraform-provider-unkey/internal/provider/models"
)

// Plan -> API
func RatelimitsToAPI(ctx context.Context, ratelimitsObj types.List) ([]components.RatelimitRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	if ratelimitsObj.IsNull() || ratelimitsObj.IsUnknown() {
		return nil, diags
	}

	var ratelimits []models.RateLimitModel
	diags.Append(ratelimitsObj.ElementsAs(ctx, &ratelimits, false)...)
	if diags.HasError() {
		return nil, diags
	}

	result := make([]components.RatelimitRequest, len(ratelimits))
	for i, rl := range ratelimits {
		autoApply := rl.AutoApply.ValueBool()
		result[i] = components.RatelimitRequest{
			Name:      rl.Name.ValueString(),
			Limit:     rl.Limit.ValueInt64(),
			Duration:  rl.Duration.ValueInt64(),
			AutoApply: &autoApply,
		}
	}
	return result, diags
}

// API -> Plan
func RatelimitsFromAPI(ctx context.Context, ratelimits []components.RatelimitResponse) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(ratelimits) == 0 {
		return types.ListNull(models.RatelimitObjectType), diags
	}

	ratelimitModels := make([]models.RateLimitModel, len(ratelimits))
	for i, rl := range ratelimits {
		ratelimitModels[i] = models.RateLimitModel{
			Name:      types.StringValue(rl.Name),
			Limit:     types.Int64Value(rl.Limit),
			Duration:  types.Int64Value(rl.Duration),
			AutoApply: types.BoolValue(rl.AutoApply),
		}
	}

	list, d := types.ListValueFrom(ctx, models.RatelimitObjectType, ratelimitModels)
	diags.Append(d...)
	return list, diags
}
