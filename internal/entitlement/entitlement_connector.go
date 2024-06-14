package entitlement

import (
	"context"
	"fmt"
)

type EntitlementConnector interface {
	// Entitlement Management
	// CreateEntitlement(ctx context.Context, entitlement Entitlement) (Entitlement, error)
	// GetEntitlementsOfSubject(ctx context.Context, subjectKey models.SubjectKey) ([]Entitlement, error)
	GetEntitlement(ctx context.Context, entitlementID NamespacedEntitlementID) (Entitlement, error)
}

type EntitlementDBConnector interface {
	// Entitlement Management
	// CreateEntitlement(ctx context.Context, entitlement Entitlement) (Entitlement, error)
	// GetEntitlementsOfSubject(ctx context.Context, subjectKey models.SubjectKey) ([]Entitlement, error)
	GetEntitlement(ctx context.Context, entitlementID NamespacedEntitlementID) (*Entitlement, error)
}

type EntitlementNotFoundError struct {
	EntitlementID NamespacedEntitlementID
}

func (e *EntitlementNotFoundError) Error() string {
	return fmt.Sprintf("entitlement not found %s in namespace %s", e.EntitlementID.ID, e.EntitlementID.Namespace)
}
