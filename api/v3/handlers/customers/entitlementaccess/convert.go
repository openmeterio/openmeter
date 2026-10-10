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

func mapEntitlementAccessCheckToAPI(access entitlement.CustomerEntitlementAccess) (api.BillingEntitlementAccessCheckResult, error) {
	result := api.BillingEntitlementAccessCheckResult{HasAccess: access.Value.HasAccess()}
	if access.Type != "" {
		result.Type = lo.ToPtr(api.BillingEntitlementType(access.Type))
	}

	switch value := access.Value.(type) {
	case *meteredentitlement.MeteredEntitlementValue, *booleanentitlement.BooleanEntitlementValue, *entitlement.NoAccessValue:
	case *staticentitlement.StaticEntitlementValue:
		result.Config = &value.Config
	default:
		return api.BillingEntitlementAccessCheckResult{}, errors.New("unknown entitlement type")
	}

	return result, nil
}

func mapEntitlementAccessToAPI(access entitlement.CustomerEntitlementAccess, expands ...api.BillingEntitlementAccessExpand) (api.BillingEntitlementValueResult, error) {
	if access.Type == "" {
		return api.BillingEntitlementValueResult{}, errors.New("entitlement type is required for value result")
	}

	result := api.BillingEntitlementValueResult{
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
	default:
		return api.BillingEntitlementValueResult{}, errors.New("unknown entitlement type")
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
