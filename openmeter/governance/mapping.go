package governance

import (
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
)

// mapEntitlementToAccess converts an entitlement value to a governance feature access result.
// When HasAccess is false, the reason code is derived from the entitlement type.
func mapEntitlementToAccess(v entitlement.EntitlementValue) FeatureAccess {
	switch ent := v.(type) {
	case *meteredentitlement.MeteredEntitlementValue:
		if ent.HasAccess() {
			return FeatureAccess{HasAccess: true}
		}
		return FeatureAccess{
			HasAccess: false,
			Reason: &AccessReason{
				Code:    ReasonUsageLimitReached,
				Message: "usage limit for feature reached",
			},
		}

	case *booleanentitlement.BooleanEntitlementValue:
		if ent.HasAccess() {
			return FeatureAccess{HasAccess: true}
		}
		return FeatureAccess{
			HasAccess: false,
			Reason: &AccessReason{
				Code:    ReasonFeatureUnavailable,
				Message: "feature is not available for customer",
			},
		}

	case *staticentitlement.StaticEntitlementValue:
		if ent.HasAccess() {
			return FeatureAccess{HasAccess: true}
		}
		return FeatureAccess{
			HasAccess: false,
			Reason: &AccessReason{
				Code:    ReasonFeatureUnavailable,
				Message: "feature is not available for customer",
			},
		}

	default:
		// NoAccessValue or unknown type
		return FeatureAccess{
			HasAccess: false,
			Reason: &AccessReason{
				Code:    ReasonFeatureUnavailable,
				Message: "feature is not available for customer",
			},
		}
	}
}
