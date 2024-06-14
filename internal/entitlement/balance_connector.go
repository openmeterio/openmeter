package entitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/streaming"
)

type EntitlementBalance struct {
	EntitlementID EntitlementID `json:"entitlementId"`
	Balance       float64       `json:"balance"`
	UsageInPeriod float64       `json:"usageInPeriod"`
	Overage       float64       `json:"overage"`
	StartOfPeriod time.Time     `json:"startOfPeriod"`
}

type EntitlementBalanceHistoryWindow struct {
	From           time.Time
	To             time.Time
	UsageInPeriod  float64
	BalanceAtStart float64
	BalanceAtEnd   float64
	Overage        float64
}

type EntitlementGrantID string

type EntitlementBalanceConnector interface {
	GetEntitlementBalance(ctx context.Context, entitlementID NamespacedEntitlementID, at time.Time) (EntitlementBalance, error)
	// GetEntitlementBalanceHistory(ctx context.Context, entitlementID NamespacedEntitlementID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, error)
	// GetEntitlementGrantBalanceHistory(ctx context.Context, entitlementGrantID EntitlementGrantID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, error)
}

type entitlementBalanceConnector struct {
	sc streaming.Connector
	oc credit.OwnerConnector
	bc credit.BalanceConnector
}

func NewEntitlementBalanceConnector(
	sc streaming.Connector,
	oc credit.OwnerConnector,
	bc credit.BalanceConnector,
) EntitlementBalanceConnector {
	return &entitlementBalanceConnector{
		sc: sc,
		oc: oc,
		bc: bc,
	}
}

func (e *entitlementBalanceConnector) GetEntitlementBalance(ctx context.Context, entitlementID NamespacedEntitlementID, at time.Time) (EntitlementBalance, error) {
	nsOwner := credit.NamespacedGrantOwner{
		Namespace: entitlementID.Namespace,
		ID:        credit.GrantOwner(entitlementID.ID),
	}
	res, err := e.bc.GetBalanceOfOwner(ctx, nsOwner, at)
	if err != nil {
		return EntitlementBalance{}, fmt.Errorf("failed to get balance of entitlement %s: %w", entitlementID.ID, err)
	}

	meterSlug, params, err := e.oc.GetOwnerQueryParams(ctx, nsOwner)
	if err != nil {
		return EntitlementBalance{}, fmt.Errorf("failed to get owner query params: %w", err)
	}

	startOfPeriod, err := e.oc.GetCurrentUsagePeriodStartAt(ctx, nsOwner, at)
	if err != nil {
		return EntitlementBalance{}, fmt.Errorf("failed to get current usage period start at: %w", err)
	}

	params.From = &startOfPeriod
	params.To = &at

	rows, err := e.sc.QueryMeter(ctx, entitlementID.Namespace, meterSlug, params)
	if err != nil {
		return EntitlementBalance{}, fmt.Errorf("failed to query meter: %w", err)
	}

	// TOOD: refactor, assert 1 row
	usage := 0.0
	for _, row := range rows {
		usage += row.Value
	}

	return EntitlementBalance{
		EntitlementID: entitlementID.ID,
		Balance:       res.Balance(),
		UsageInPeriod: usage,
		Overage:       res.Overage,
		StartOfPeriod: startOfPeriod,
	}, nil
}
