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
func mapEntitlementValueToAPI(featureKey string, entitlementValue entitlement.EntitlementValue) (api.BillingEntitlementAccessResult, error) {
	switch ent := entitlementValue.(type) {
	case *meteredentitlement.MeteredEntitlementValue:
		return api.BillingEntitlementAccessResult{
			FeatureKey: featureKey,
			Type:       api.BillingEntitlementTypeMetered,
			HasAccess:  ent.HasAccess(),
		}, nil
	case *staticentitlement.StaticEntitlementValue:
		var jsonValue interface{}

		if ent.Config != nil {
			// FIXME (pmarton): static config is double json encoded, we need to unmarshal it twice
			var inner string
			if err := json.Unmarshal([]byte(ent.Config), &inner); err != nil {
				return api.BillingEntitlementAccessResult{}, models.NewGenericValidationError(
					fmt.Errorf("failed to unmarshal static entitlement config: %w", err),
				)
			}

			// Return config as JSON value
			if err := json.Unmarshal([]byte(inner), &jsonValue); err != nil {
				return api.BillingEntitlementAccessResult{}, models.NewGenericValidationError(
					fmt.Errorf("failed to unmarshal static entitlement config: %w", err),
				)
			}
		}

		return api.BillingEntitlementAccessResult{
			FeatureKey: featureKey,
			Type:       api.BillingEntitlementTypeStatic,
			HasAccess:  ent.HasAccess(),
			Config:     &jsonValue,
		}, nil
	case *booleanentitlement.BooleanEntitlementValue:
		return api.BillingEntitlementAccessResult{
			FeatureKey: featureKey,
			Type:       api.BillingEntitlementTypeBoolean,
			HasAccess:  ent.HasAccess(),
		}, nil

	case *entitlement.NoAccessValue:
		// FIXME(pmarton): do we need to handle no access value?
		// According to comments this only happens when the entitlement is not active
		return api.BillingEntitlementAccessResult{}, errors.New("entitlement is not active")
	default:
		return api.BillingEntitlementAccessResult{}, errors.New("unknown entitlement type")
	}
}
