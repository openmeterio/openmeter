package entitlement

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Connector interface {
	// Entitlement Management
	CreateEntitlement(ctx context.Context, entitlement Entitlement) (Entitlement, error)
	GetEntitlementsOfSubject(ctx context.Context, subjectKey models.SubjectKey) ([]Entitlement, error)

	// Balance & Usage
	// GetEntitlementBalance(ctx context.Context, entitlementID models.NamespacedID) (EntitlementBalance, error)
	// GetEntitlementBalanceHistory(ctx context.Context, entitlementID models.NamespacedID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, error)
	// ResetEntitlementUsage(ctx context.Context, entitlementID models.NamespacedID) error

	GetEntitlementGrantBalanceHistory(ctx context.Context, entitlementGrantID EntitlementGrantID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, error)

	// Reset Scheduling
	ChangeEntitlementUsageResetSchedule(ctx context.Context, entitlementID models.NamespacedID, schedule Schedule) (Schedule, error)
}

type Schedule interface{}
