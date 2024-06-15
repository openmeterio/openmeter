package entitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type EntitlementDBConnector interface {
	// Entitlement Management
	// GetEntitlementsOfSubject(ctx context.Context, subjectKey models.SubjectKey) ([]Entitlement, error)
	CreateEntitlement(ctx context.Context, entitlement CreateEntitlementInputs) (*Entitlement, error)
	GetEntitlement(ctx context.Context, entitlementID models.NamespacedID) (*Entitlement, error)
}

type UsageResetDBConnector interface {
	Save(ctx context.Context, usageResetTime UsageResetTime) error
	GetLastAt(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (*UsageResetTime, error)
	GetBetween(ctx context.Context, entitlementID models.NamespacedID, from time.Time, to time.Time) ([]UsageResetTime, error)
}

type UsageResetNotFoundError struct {
	EntitlementID models.NamespacedID
}

func (e UsageResetNotFoundError) Error() string {
	return fmt.Sprintf("usage reset not found for entitlement %s in namespace %s", e.EntitlementID.ID, e.EntitlementID.Namespace)
}
