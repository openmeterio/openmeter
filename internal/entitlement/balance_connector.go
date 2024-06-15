package entitlement

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type EntitlementBalance struct {
	EntitlementID string    `json:"entitlementId"`
	Balance       float64   `json:"balance"`
	UsageInPeriod float64   `json:"usageInPeriod"`
	Overage       float64   `json:"overage"`
	StartOfPeriod time.Time `json:"startOfPeriod"`
}

type EntitlementBalanceHistoryWindow struct {
	From           time.Time
	To             time.Time
	UsageInPeriod  float64
	BalanceAtStart float64
	OverageAtStart float64
}

type EntitlementGrantID string

type WindowSize string

const (
	// We don't support minute precision as that results in an extremely heavy calculation
	// WindowSizeMinute WindowSize = "MINUTE"
	WindowSizeHour WindowSize = "HOUR"
	WindowSizeDay  WindowSize = "DAY"
)

type BalanceHistoryParams struct {
	From           time.Time
	To             time.Time
	WindowSize     WindowSize
	WindowTimeZone time.Location
}

type EntitlementBalanceConnector interface {
	GetEntitlementBalance(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (EntitlementBalance, error)
	GetEntitlementBalanceHistory(ctx context.Context, entitlementID models.NamespacedID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, error)

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

func (e *entitlementBalanceConnector) GetEntitlementBalance(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (EntitlementBalance, error) {
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

	startOfPeriod, err := e.oc.GetUsagePeriodStartAt(ctx, nsOwner, at)
	if err != nil {
		return EntitlementBalance{}, fmt.Errorf("failed to get current usage period start at: %w", err)
	}

	params.From = &startOfPeriod
	params.To = &at

	rows, err := e.sc.QueryMeter(ctx, entitlementID.Namespace, meterSlug, params)
	if err != nil {
		return EntitlementBalance{}, fmt.Errorf("failed to query meter: %w", err)
	}

	// TODO: refactor, assert 1 row
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

func (e *entitlementBalanceConnector) GetEntitlementBalanceHistory(ctx context.Context, entitlementID models.NamespacedID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, error) {
	// TODO: we should guard against abuse, getting history is expensive

	owner := credit.NamespacedGrantOwner{
		Namespace: entitlementID.Namespace,
		ID:        credit.GrantOwner(entitlementID.ID),
	}

	// 1. we get the burndown history
	burndownHistory, err := e.bc.GetBalanceHistoryOfOwner(ctx, owner, credit.BalanceHistoryParams{
		From: params.From,
		To:   params.To,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get balance history: %w", err)
	}
	// 2. and we get the windowed usage data
	meterSlug, meterParams, err := e.oc.GetOwnerQueryParams(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner query params: %w", err)
	}
	meterParams.From = &params.From
	meterParams.To = &params.To
	meterParams.WindowSize = convert.ToPointer(models.WindowSize(params.WindowSize))
	meterParams.WindowTimeZone = &params.WindowTimeZone

	meterRows, err := e.sc.QueryMeter(ctx, owner.Namespace, meterSlug, meterParams)
	if err != nil {
		return nil, fmt.Errorf("failed to query meter: %w", err)
	}
	// 3. and then we merge the two

	// convert history segments to list of point in time balances
	segments := burndownHistory.Segments()

	if len(segments) == 0 {
		return nil, fmt.Errorf("returned history is empty")
	}

	timestampedBalances := make([]struct {
		balance   float64
		overage   float64
		timestamp time.Time
	}, 0, len(segments))
	for _, segment := range segments {
		timestampedBalances = append(timestampedBalances, struct {
			balance   float64
			overage   float64
			timestamp time.Time
		}{
			balance:   segment.BalanceAtStart.Balance(),
			overage:   segment.Overage,
			timestamp: segment.Period.From,
		})
	}

	// we'll create a window for each row (same windowsize)
	windows := make([]EntitlementBalanceHistoryWindow, 0, len(meterRows))
	for _, row := range meterRows {

		// find the last timestamped balance that was not later than the row
		// This is not effective on a lot of rows
		tsBalance, ok := slicesx.First(timestampedBalances, func(tsb struct {
			balance   float64
			overage   float64
			timestamp time.Time
		}) bool {
			return tsb.timestamp.Before(row.WindowStart) || tsb.timestamp.Equal(row.WindowStart)
		}, true)
		if !ok {
			return nil, fmt.Errorf("no balance found for time %s", row.WindowStart.Format(time.RFC3339))
		}

		window := EntitlementBalanceHistoryWindow{
			From:           row.WindowStart.In(&params.WindowTimeZone),
			To:             row.WindowEnd.In(&params.WindowTimeZone),
			UsageInPeriod:  row.Value,
			BalanceAtStart: tsBalance.balance,
			OverageAtStart: tsBalance.overage,
		}
		windows = append(windows, window)

		// FIXME: clean up these calculations
		// deduct balance and increase overage if needed
		usage := row.Value

		balanceAtEnd := math.Max(0, tsBalance.balance-usage)
		deductedUsage := tsBalance.balance - balanceAtEnd
		overage := usage - deductedUsage + tsBalance.overage

		// update
		tsBalance.balance = balanceAtEnd
		tsBalance.overage = overage
	}

	return windows, nil
}
