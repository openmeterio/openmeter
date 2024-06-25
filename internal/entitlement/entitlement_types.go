package entitlement

import (
	"time"

	"github.com/openmeterio/openmeter/internal/productcatalog"
)

type EntitlementValue interface {
	HasAccess() bool
}

type SubTypeConnector interface {
	GetValue(entitlement *Entitlement, at time.Time) (EntitlementValue, error)
	SetDefaultsAndValidate(entitlement *CreateEntitlementInputs) error

	// ValidateForFeature validates the entitlement against the feature.
	ValidateForFeature(entitlement *CreateEntitlementInputs, feature productcatalog.Feature) error
}
