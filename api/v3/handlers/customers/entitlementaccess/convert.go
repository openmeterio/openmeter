package customersentitlement

import (
	"errors"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
)

func mapEntitlementAccessCheckToAPI(access entitlement.CustomerEntitlementAccess) (apiv3models.BillingEntitlementAccessCheckResult, error) {
	result := apiv3models.BillingEntitlementAccessCheckResult{HasAccess: access.Value.HasAccess()}
	if access.Type != "" {
		result.Type = lo.ToPtr(apiv3models.BillingEntitlementType(access.Type))
	}

	switch value := access.Value.(type) {
	case *meteredentitlement.MeteredEntitlementValue, *booleanentitlement.BooleanEntitlementValue, *entitlement.NoAccessValue:
	case *staticentitlement.StaticEntitlementValue:
		result.Config = &value.Config
	default:
		return apiv3models.BillingEntitlementAccessCheckResult{}, errors.New("unknown entitlement type")
	}

	return result, nil
}

func mapEntitlementAccessToAPI(access entitlement.CustomerEntitlementAccess, expands ...apiv3models.BillingEntitlementAccessExpand) (apiv3models.BillingEntitlementValueResult, error) {
	if access.Type == "" {
		return apiv3models.BillingEntitlementValueResult{}, errors.New("entitlement type is required for value result")
	}

	result := apiv3models.BillingEntitlementValueResult{
		FeatureKey: access.FeatureKey,
		Type:       apiv3models.BillingEntitlementType(access.Type),
		HasAccess:  access.Value.HasAccess(),
	}

	switch value := access.Value.(type) {
	case *meteredentitlement.MeteredEntitlementValue:
		if lo.Contains(expands, apiv3models.BillingEntitlementAccessExpandValue) {
			result.Value = lo.ToPtr(mapMeteredEntitlementValueToAPI(value))
		}
	case *staticentitlement.StaticEntitlementValue:
		result.Config = &value.Config
	case *booleanentitlement.BooleanEntitlementValue:
	case *entitlement.NoAccessValue:
	default:
		return apiv3models.BillingEntitlementValueResult{}, errors.New("unknown entitlement type")
	}

	return result, nil
}

func mapMeteredEntitlementValueToAPI(value *meteredentitlement.MeteredEntitlementValue) apiv3models.BillingEntitlementAccessValue {
	grantBalances := make(map[string]apiv3models.Numeric, len(value.GrantBalances))
	for grantID, balance := range value.GrantBalances {
		grantBalances[grantID] = alpacadecimal.NewFromFloat(balance).String()
	}

	return apiv3models.BillingEntitlementAccessValue{
		Balance:                   alpacadecimal.NewFromFloat(value.Balance).String(),
		Usage:                     alpacadecimal.NewFromFloat(value.UsageInPeriod).String(),
		Overage:                   alpacadecimal.NewFromFloat(value.Overage).String(),
		TotalAvailableGrantAmount: alpacadecimal.NewFromFloat(value.TotalAvailableGrantAmount).String(),
		GrantBalances:             grantBalances,
	}
}
