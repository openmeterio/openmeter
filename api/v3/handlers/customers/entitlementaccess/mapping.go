package customersentitlement

import (
	"encoding/json"
	"errors"
	"fmt"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/pkg/models"
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

		// If config is not nil, unmarshal it
		if ent.Config != nil {
			var jsonValue interface{}

			// FIXME (pmarton): static config is double json encoded, we need to unmarshal it twice
			var inner string
			if err := json.Unmarshal(ent.Config, &inner); err != nil {
				return true, api.BillingEntitlementAccessResult{}, models.NewGenericValidationError(
					fmt.Errorf("failed to unmarshal static entitlement config: %w", err),
				)
			}

			// Return config as JSON value
			if err := json.Unmarshal([]byte(inner), &jsonValue); err != nil {
				return true, api.BillingEntitlementAccessResult{}, models.NewGenericValidationError(
					fmt.Errorf("failed to unmarshal static entitlement config: %w", err),
				)
			}

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
