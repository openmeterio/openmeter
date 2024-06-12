package entitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Connector interface {
	// Entitlement Management
	CreateEntitlement(ctx context.Context, entitlement Entitlement) (Entitlement, error)
	GetEntitlementsOfSubject(ctx context.Context, subjectKey models.SubjectKey) ([]Entitlement, error)

	// Balance & Usage
	GetEntitlementBalance(ctx context.Context, entitlementID NamespacedEntitlementID) (EntitlementBalance, error)
	GetEntitlementBalanceHistory(ctx context.Context, entitlementID NamespacedEntitlementID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, error)
	GetEntitlementGrantBalanceHistory(ctx context.Context, entitlementGrantID EntitlementGrantID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, error)

	ResetEntitlementUsage(ctx context.Context, entitlementID NamespacedEntitlementID) error

	// Reset Scheduling
	ChangeEntitlementUsageResetSchedule(ctx context.Context, entitlementID NamespacedEntitlementID, schedule Schedule) (Schedule, error)
}

type BalanceHistoryParams struct {
	From           time.Time
	To             time.Time
	WindowSize     models.WindowSize
	WindowTimeZone time.Location
}

type Schedule interface{}
