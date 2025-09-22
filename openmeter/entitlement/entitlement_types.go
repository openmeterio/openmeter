package entitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

type EntitlementValue interface {
	HasAccess() bool
}

var _ EntitlementValue = (*NoAccessValue)(nil)

type NoAccessValue struct{}

func (*NoAccessValue) HasAccess() bool {
	return false
}

// FIXME[galexi]: we can get rid of this concept due to better hierarchy
type SubTypeConnector interface {
	GetValue(ctx context.Context, entitlement *Entitlement, at time.Time) (EntitlementValue, error)

	// Runs before creating the entitlement, building the Repository inputs.
	// If it returns an error the operation has to fail.
	BeforeCreate(entitlement CreateEntitlementInputs, feature feature.Feature) (*CreateEntitlementRepoInputs, error)

	// Runs after entitlement creation.
	// If it returns an error the operation has to fail.
	AfterCreate(ctx context.Context, entitlement *Entitlement) error
}
