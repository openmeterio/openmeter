package service

import (
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/governance"
)

// mapEntitlementToAccess converts an entitlement value to a governance feature access result.
// When HasAccess is false, the reason code is derived from the entitlement type.
func mapEntitlementToAccess(v entitlement.EntitlementValue) governance.FeatureAccess {
	switch ent := v.(type) {
	case *meteredentitlement.MeteredEntitlementValue:
		if ent.HasAccess() {
			return governance.FeatureAccess{HasAccess: true}
		}
		return governance.FeatureAccess{
			HasAccess: false,
			Reason: &governance.AccessReason{
				Code:    governance.ReasonUsageLimitReached,
				Message: "usage limit for feature reached",
			},
		}

	case *booleanentitlement.BooleanEntitlementValue:
		if ent.HasAccess() {
			return governance.FeatureAccess{HasAccess: true}
		}
		return governance.FeatureAccess{
			HasAccess: false,
			Reason: &governance.AccessReason{
				Code:    governance.ReasonFeatureUnavailable,
				Message: "feature is not available for customer",
			},
		}

	case *staticentitlement.StaticEntitlementValue:
		if ent.HasAccess() {
			return governance.FeatureAccess{HasAccess: true}
		}
		return governance.FeatureAccess{
			HasAccess: false,
			Reason: &governance.AccessReason{
				Code:    governance.ReasonFeatureUnavailable,
				Message: "feature is not available for customer",
			},
		}

	default:
		// NoAccessValue or unknown type
		return governance.FeatureAccess{
			HasAccess: false,
			Reason: &governance.AccessReason{
				Code:    governance.ReasonFeatureUnavailable,
				Message: "feature is not available for customer",
			},
		}
	}
}
