package conversions

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/unkeyed/sdks/api/go/v2/models/components"

	"github.com/TeamEyesoft/terraform-provider-unkey/internal/provider/models"
)

// Plan -> API
func CreditsToAPI(ctx context.Context, creditsObj types.Object) (*components.KeyCreditsData, diag.Diagnostics) {
	var diags diag.Diagnostics
	if creditsObj.IsNull() || creditsObj.IsUnknown() {
		return nil, diags
	}

	var credits models.KeyCreditsModel
	diags.Append(creditsObj.As(ctx, &credits, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	var refillData *components.KeyCreditsRefill
	if !credits.Refill.IsNull() && !credits.Refill.IsUnknown() {
		var refill models.KeyCreditsRefillModel
		diags.Append(credits.Refill.As(ctx, &refill, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			refillData = &components.KeyCreditsRefill{
				Interval:  components.KeyCreditsRefillInterval(refill.Interval.ValueString()),
				Amount:    refill.Amount.ValueInt64(),
				RefillDay: refill.RefillDay.ValueInt64Pointer(),
			}
		}
	}

	return &components.KeyCreditsData{
		Remaining: credits.Remaining.ValueInt64Pointer(),
		Refill:    refillData,
	}, diags
}

func CreditsToUpdateAPI(ctx context.Context, creditsObj types.Object) (*components.UpdateKeyCreditsData, diag.Diagnostics) {
	keyCredits, diags := CreditsToAPI(ctx, creditsObj)

	return &components.UpdateKeyCreditsData{
		Remaining: keyCredits.Remaining,
		Refill: &components.UpdateKeyCreditsRefill{
			Interval:  components.UpdateKeyCreditsRefillInterval(keyCredits.Refill.Interval),
			Amount:    keyCredits.Refill.Amount,
			RefillDay: keyCredits.Refill.RefillDay,
		},
	}, diags
}

// API -> Plan
func CreditsFromAPI(ctx context.Context, credits *components.KeyCreditsData) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	if credits == nil {
		return types.ObjectNull(models.CreditsAttrTypes), diags
	}

	var refillValue types.Object
	if credits.Refill != nil {
		refillModel := models.KeyCreditsRefillModel{
			Interval:  types.StringValue(string(credits.Refill.Interval)),
			Amount:    types.Int64Value(credits.Refill.Amount),
			RefillDay: types.Int64PointerValue(credits.Refill.RefillDay),
		}
		refillValue, diags = types.ObjectValueFrom(ctx, models.CreditsRefillAttrTypes, refillModel)
	} else {
		refillValue = types.ObjectNull(models.CreditsRefillAttrTypes)
	}

	creditsModel := models.KeyCreditsModel{
		Remaining: types.Int64PointerValue(credits.Remaining),
		Refill:    refillValue,
	}

	creditsObj, d := types.ObjectValueFrom(ctx, models.CreditsAttrTypes, creditsModel)
	diags.Append(d...)
	return creditsObj, diags
}
