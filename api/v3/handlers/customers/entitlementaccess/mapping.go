package customersentitlement

import (
	"errors"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
)

// mapEntitlementValueToAPI maps an entitlement value to an API entitlement access result.
func mapEntitlementValueToAPI(featureKey string, entitlementValue entitlement.EntitlementValue) (bool, api.BillingEntitlementAccessResult, error) {
	switch ent := entitlementValue.(type) {
	case *meteredentitlement.MeteredEntitlementValue:
		return true, api.BillingEntitlementAccessResult{
			FeatureKey: featureKey,
			Type:       api.BillingEntitlementTypeMetered,
			HasAccess:  ent.HasAccess(),
		}, nil
	case *staticentitlement.StaticEntitlementValue:
		accessResult := api.BillingEntitlementAccessResult{
			FeatureKey: featureKey,
			Type:       api.BillingEntitlementTypeStatic,
			HasAccess:  ent.HasAccess(),
		}

		// Config is now properly encoded (unwrapped at DB layer)
		if ent.Config != nil {
			jsonValue := string(ent.Config)
			accessResult.Config = &jsonValue
		}

		return true, accessResult, nil
	case *booleanentitlement.BooleanEntitlementValue:
		return true, api.BillingEntitlementAccessResult{
			FeatureKey: featureKey,
			Type:       api.BillingEntitlementTypeBoolean,
			HasAccess:  ent.HasAccess(),
		}, nil

	case *entitlement.NoAccessValue:
		// FIXME(pmarton): do we need to handle no access value?
		// According to comments this only happens when the entitlement is not active
		// This is the reason why we return a bool and not an error
		return false, api.BillingEntitlementAccessResult{}, nil
	default:
		return true, api.BillingEntitlementAccessResult{}, errors.New("unknown entitlement type")
	}
}
