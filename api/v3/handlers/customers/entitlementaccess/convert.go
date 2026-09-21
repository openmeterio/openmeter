package customersentitlement

import (
	"errors"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
)

func mapEntitlementAccessToAPI(access entitlement.CustomerEntitlementAccess, expands ...api.BillingEntitlementAccessExpand) (api.BillingEntitlementAccessResult, error) {
	result := api.BillingEntitlementAccessResult{
		FeatureKey: access.FeatureKey,
		Type:       api.BillingEntitlementType(access.Type),
		HasAccess:  access.Value.HasAccess(),
	}

	switch value := access.Value.(type) {
	case *meteredentitlement.MeteredEntitlementValue:
		if lo.Contains(expands, api.BillingEntitlementAccessExpandValue) {
			result.Value = lo.ToPtr(mapMeteredEntitlementValueToAPI(value))
		}
	case *staticentitlement.StaticEntitlementValue:
		result.Config = &value.Config
	case *booleanentitlement.BooleanEntitlementValue:
	case *entitlement.NoAccessValue:
		// A feature without any entitlement has no type, but the contract requires one.
		if access.Type == "" {
			result.Type = api.BillingEntitlementTypeStatic
		}
	default:
		return api.BillingEntitlementAccessResult{}, errors.New("unknown entitlement type")
	}

	return result, nil
}

func mapMeteredEntitlementValueToAPI(value *meteredentitlement.MeteredEntitlementValue) api.BillingEntitlementAccessValue {
	grantBalances := make(map[string]api.Numeric, len(value.GrantBalances))
	for grantID, balance := range value.GrantBalances {
		grantBalances[grantID] = alpacadecimal.NewFromFloat(balance).String()
	}

	return api.BillingEntitlementAccessValue{
		Balance:                   alpacadecimal.NewFromFloat(value.Balance).String(),
		Usage:                     alpacadecimal.NewFromFloat(value.UsageInPeriod).String(),
		Overage:                   alpacadecimal.NewFromFloat(value.Overage).String(),
		TotalAvailableGrantAmount: alpacadecimal.NewFromFloat(value.TotalAvailableGrantAmount).String(),
		GrantBalances:             grantBalances,
	}
}
