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
	featureKey := access.FeatureKey

	switch ent := access.Value.(type) {
	case *meteredentitlement.MeteredEntitlementValue:
		result := api.BillingEntitlementAccessResult{
			FeatureKey: featureKey,
			Type:       api.BillingEntitlementTypeMetered,
			HasAccess:  ent.HasAccess(),
		}

		if lo.Contains(expands, api.BillingEntitlementAccessExpandValue) {
			result.Value = lo.ToPtr(mapMeteredEntitlementValueToAPI(ent))
		}

		return result, nil
	case *staticentitlement.StaticEntitlementValue:
		return api.BillingEntitlementAccessResult{
			FeatureKey: featureKey,
			Type:       api.BillingEntitlementTypeStatic,
			HasAccess:  ent.HasAccess(),
			Config:     &ent.Config,
		}, nil
	case *booleanentitlement.BooleanEntitlementValue:
		return api.BillingEntitlementAccessResult{
			FeatureKey: featureKey,
			Type:       api.BillingEntitlementTypeBoolean,
			HasAccess:  ent.HasAccess(),
		}, nil
	case *entitlement.NoAccessValue:
		return api.BillingEntitlementAccessResult{
			HasAccess:  false,
			FeatureKey: featureKey,
			// using a constant value to satisfy the API contract
			Type: api.BillingEntitlementTypeStatic,
		}, nil
	default:
		return api.BillingEntitlementAccessResult{}, errors.New("unknown entitlement type")
	}
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
