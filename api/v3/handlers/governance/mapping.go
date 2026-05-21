package governance

import (
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
)

// mapEntitlementToAccess converts an entitlement value to a governance feature access result.
// When has_access is false, the reason code is derived from the entitlement type.
func mapEntitlementToAccess(v entitlement.EntitlementValue) api.GovernanceFeatureAccess {
	switch ent := v.(type) {
	case *meteredentitlement.MeteredEntitlementValue:
		if ent.HasAccess() {
			return api.GovernanceFeatureAccess{HasAccess: true}
		}
		return api.GovernanceFeatureAccess{
			HasAccess: false,
			Reason: &api.GovernanceFeatureAccessReason{
				Code:    api.GovernanceFeatureAccessReasonCodeUsageLimitReached,
				Message: "usage limit for feature reached",
			},
		}

	case *booleanentitlement.BooleanEntitlementValue:
		if ent.HasAccess() {
			return api.GovernanceFeatureAccess{HasAccess: true}
		}
		return api.GovernanceFeatureAccess{
			HasAccess: false,
			Reason: &api.GovernanceFeatureAccessReason{
				Code:    api.GovernanceFeatureAccessReasonCodeFeatureUnavailable,
				Message: "feature is not available for customer",
			},
		}

	case *staticentitlement.StaticEntitlementValue:
		if ent.HasAccess() {
			return api.GovernanceFeatureAccess{HasAccess: true}
		}
		return api.GovernanceFeatureAccess{
			HasAccess: false,
			Reason: &api.GovernanceFeatureAccessReason{
				Code:    api.GovernanceFeatureAccessReasonCodeFeatureUnavailable,
				Message: "feature is not available for customer",
			},
		}

	default:
		// NoAccessValue or unknown type
		return api.GovernanceFeatureAccess{
			HasAccess: false,
			Reason: &api.GovernanceFeatureAccessReason{
				Code:    api.GovernanceFeatureAccessReasonCodeFeatureUnavailable,
				Message: "feature is not available for customer",
			},
		}
	}
}
