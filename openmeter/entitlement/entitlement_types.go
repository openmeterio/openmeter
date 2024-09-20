package entitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

type EntitlementValue interface {
	HasAccess() bool
}

type SubTypeConnector interface {
	GetValue(entitlement *Entitlement, at time.Time) (EntitlementValue, error)

	// Runs before creating the entitlement, building the Repository inputs.
	// If it returns an error the operation has to fail.
	BeforeCreate(entitlement CreateEntitlementInputs, feature feature.Feature) (*CreateEntitlementRepoInputs, error)

	// Runs after entitlement creation.
	// If it returns an error the operation has to fail.
	AfterCreate(ctx context.Context, entitlement *Entitlement) error
}
