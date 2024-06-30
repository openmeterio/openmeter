package entitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/internal/productcatalog"
)

type EntitlementValue interface {
	HasAccess() bool
}

type SubTypeConnector interface {
	GetValue(entitlement *Entitlement, at time.Time) (EntitlementValue, error)

	// Runs before creating the entitlement. Might manipulate the inputs.
	// If it returns an error the operation has to fail.
	BeforeCreate(entitlement *CreateEntitlementInputs, feature *productcatalog.Feature) error

	// Runs after entitlement creation.
	// If it returns an error the operation has to fail.
	AfterCreate(ctx context.Context, entitlement *Entitlement) error
}
