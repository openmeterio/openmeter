package meteredentitlement

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type EntitlementBalance struct {
	EntitlementID             string    `json:"entitlementId"`
	Balance                   float64   `json:"balance"`
	UsageInPeriod             float64   `json:"usageInPeriod"`
	Overage                   float64   `json:"overage"`
	TotalAvailableGrantAmount float64   `json:"totalAvailableGrantAmount"`
	StartOfPeriod             time.Time `json:"startOfPeriod"`
}

type EntitlementBalanceHistoryWindow struct {
	From           time.Time
	To             time.Time
	UsageInPeriod  float64
	BalanceAtStart float64
	OverageAtStart float64
}

type WindowSize string

const (
	// We don't support minute precision as that results in an extremely heavy calculation
	// WindowSizeMinute WindowSize = "MINUTE"
	WindowSizeHour WindowSize = "HOUR"
	WindowSizeDay  WindowSize = "DAY"
)

type BalanceHistoryParams struct {
	From           *time.Time
	To             *time.Time
	WindowSize     WindowSize
	WindowTimeZone time.Location
}

func (e *connector) GetEntitlementBalance(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (*EntitlementBalance, error) {
	ctx, span := e.tracer.Start(ctx, "meteredentitlement.GetEntitlementBalance", trace.WithAttributes(
		attribute.String("entitlement_id", entitlementID.ID),
		attribute.String("at", at.Format(time.RFC3339)),
	))
	defer span.End()

	e.logger.DebugContext(ctx, "Getting entitlement balance", "entitlement", entitlementID, "at", at)

	// We round up to closest full minute to include all the partial usage in the last minute of querying
	// Not that this will never throw us to a different usage period
	if trunc := at.Truncate(time.Minute); trunc.Before(at) {
		at = trunc.Add(time.Minute)
	}

	nsOwner := models.NamespacedID{
		Namespace: entitlementID.Namespace,
		ID:        entitlementID.ID,
	}

	startOfPeriod, err := e.ownerConnector.GetUsagePeriodStartAt(ctx, nsOwner, at)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage period start at: %w", err)
	}

	// Let's calculate balance since the last snapshot
	res, err := e.balanceConnector.GetBalanceAt(ctx, nsOwner, at)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance since snapshot: %w", err)
	}

	return &EntitlementBalance{
		EntitlementID:             entitlementID.ID,
		Balance:                   res.Snapshot.Balance(),
		UsageInPeriod:             res.Snapshot.Usage.Usage,
		Overage:                   res.Snapshot.Overage,
		TotalAvailableGrantAmount: res.TotalAvailableGrantAmount(),
		StartOfPeriod:             startOfPeriod,
	}, nil
}

func (e *connector) GetEntitlementBalanceHistory(ctx context.Context, entitlementID models.NamespacedID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, engine.GrantBurnDownHistory, error) {
	ctx, span := e.tracer.Start(ctx, "meteredentitlement.GetEntitlementBalanceHistory")
	defer span.End()

	// TODO: we should guard against abuse, getting history is expensive

	// validate that we're working with a metered entitlement
	entRepoEntity, err := e.entitlementRepo.GetEntitlement(ctx, entitlementID)
	if err != nil {
		return nil, engine.GrantBurnDownHistory{}, err
	}

	if entRepoEntity == nil {
		return nil, engine.GrantBurnDownHistory{}, &entitlement.NotFoundError{EntitlementID: entitlementID}
	}

	ent, err := ParseFromGenericEntitlement(entRepoEntity)
	if err != nil {
		return nil, engine.GrantBurnDownHistory{}, err
	}

	if params.From == nil {
		params.From = &ent.LastReset
	}

	if params.To == nil {
		params.To = convert.ToPointer(clock.Now())
	}

	// query period cannot be before start of measuring usage
	if params.From.Before(ent.MeasureUsageFrom) {
		return nil, engine.GrantBurnDownHistory{}, models.NewGenericValidationError(fmt.Errorf("from %s cannot be before %s", params.From.UTC().Format(time.RFC3339), ent.MeasureUsageFrom.UTC().Format(time.RFC3339)))
	}

	owner, err := e.ownerConnector.DescribeOwner(ctx, models.NamespacedID{
		Namespace: entitlementID.Namespace,
		ID:        entitlementID.ID,
	})
	if err != nil {
		return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("failed to describe owner: %w", err)
	}

	// FIXME: remove truncation
	fullPeriodTruncated := timeutil.ClosedPeriod{
		From: params.From.Truncate(time.Minute),
		To:   params.To.Truncate(time.Minute),
	}

	// If `to` time is not truncated to minute we assume to query until the next minute so fresh usage data shows up
	if !params.To.Truncate(time.Minute).Equal(*params.To) {
		fullPeriodTruncated.To = fullPeriodTruncated.To.Add(time.Minute)
	}

	// 1. Let's query the windowed usage data
	getBaseQuery := func() streaming.QueryParams {
		base := owner.DefaultQueryParams

		base.From = convert.ToPointer(fullPeriodTruncated.From)
		base.To = convert.ToPointer(fullPeriodTruncated.To)
		base.WindowSize = convert.ToPointer(meter.WindowSize(params.WindowSize))
		base.WindowTimeZone = &params.WindowTimeZone

		return base
	}

	// 2. and we get the windowed usage data
	meterRows, err := e.queryMeter(ctx, owner.Namespace, owner.Meter, getBaseQuery())
	if err != nil {
		return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("failed to query meter: %w", err)
	}

	// If we get 0 rows that means the windowsize is larger than the queried period.
	// In this case we simply query for the entire period.
	if len(meterRows) == 0 {
		nonWindowedParams := getBaseQuery()
		nonWindowedParams.WindowSize = nil
		nonWindowedParams.WindowTimeZone = nil
		meterRows, err = e.queryMeter(ctx, owner.NamespacedID.Namespace, owner.Meter, nonWindowedParams)
		if err != nil {
			return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("failed to query meter: %w", err)
		}
	}

	// Clickhouse only returns rows where there is usage data. The history response needs to contain each window even if there is no usage data, so we need to fill in the missing windows with 0 usage

	// We need to truncate to the window size
	startOfFirstWindowThatShouldBePresent, err := meter.WindowSize(params.WindowSize).Truncate(fullPeriodTruncated.From)
	if err != nil {
		return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("failed to truncate to window size: %w", err)
	}

	endOfLastWindowThatShouldBePresent, err := meter.WindowSize(params.WindowSize).Truncate(fullPeriodTruncated.To)
	if err != nil {
		return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("failed to truncate to window size: %w", err)
	}

	// If we did truncate we need to advance one window size to the right so the original period.To is included
	if !endOfLastWindowThatShouldBePresent.Equal(fullPeriodTruncated.To) {
		endOfLastWindowThatShouldBePresent, err = meter.WindowSize(params.WindowSize).AddTo(endOfLastWindowThatShouldBePresent)
		if err != nil {
			return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("failed to add window size: %w", err)
		}
	}

	// Now, let's fill the missing windows
	allRows := make([]meter.MeterQueryRow, 0)

	// by this point we know we can add params.WindowSize
	for current := startOfFirstWindowThatShouldBePresent; current.Before(endOfLastWindowThatShouldBePresent); current, _ = meter.WindowSize(params.WindowSize).AddTo(current) {
		wEnd, _ := meter.WindowSize(params.WindowSize).AddTo(current)
		row := meter.MeterQueryRow{
			WindowStart: current,
			WindowEnd:   wEnd,
		}

		// Let's see if there's a matching row in meterRows
		matchingRow, ok := lo.Find(meterRows, func(row meter.MeterQueryRow) bool {
			return row.WindowStart.Equal(current) && row.WindowEnd.Equal(wEnd)
		})

		if ok {
			row.Value = matchingRow.Value
			row.Subject = matchingRow.Subject
			row.GroupBy = matchingRow.GroupBy
		}

		allRows = append(allRows, row)
	}

	// Due to windowing, it is possible that a window is before the entitlement's startOfMeasurement, we need to filter these out
	allRows = lo.Filter(allRows, func(row meter.MeterQueryRow, _ int) bool {
		return !row.WindowStart.Before(ent.MeasureUsageFrom)
	})

	// 2. Let's get the history for the period
	periodToQueryEngine := fullPeriodTruncated

	if len(allRows) > 0 {
		// If the window starts earlier than our period, we need to query the engine starting from the window start
		if allRows[0].WindowStart.Before(fullPeriodTruncated.From) {
			periodToQueryEngine.From = allRows[0].WindowStart
		}
	}

	historyRes, err := e.balanceConnector.GetBalanceForPeriod(ctx, owner.NamespacedID, timeutil.ClosedPeriod{
		From: periodToQueryEngine.From,
		To:   periodToQueryEngine.To,
	})
	if err != nil {
		return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("failed to get balance history: %w", err)
	}

	// convert history segments to list of point-in-time balances
	segments := historyRes.History.Segments()

	if len(segments) == 0 {
		return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("returned history is empty")
	}

	// We'll use these balances to continuously deduct usage from
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
			timestamp: segment.ClosedPeriod.From,
		})
	}

	// 3. and then we merge the two

	// we'll create a window for each row (same windowsize)
	windows := make([]EntitlementBalanceHistoryWindow, 0, len(allRows))
	visited := make(map[int]bool)

	for _, row := range allRows {
		// Lets find the last timestamped balance that was no later than the row
		tsBalance, idx, ok := slicesx.Last(timestampedBalances, func(tsb struct {
			balance   float64
			overage   float64
			timestamp time.Time
		},
		) bool {
			return tsb.timestamp.Before(row.WindowStart) || tsb.timestamp.Equal(row.WindowStart)
		})
		if !ok {
			return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("no balance found for time %s", row.WindowStart.Format(time.RFC3339))
		}

		// If this is the first time we're using this `tsBalance`, we need to account for the usage between it's time and the row's time
		if !visited[idx] {
			// We need to query the usage between the two timestamps
			params := getBaseQuery()
			params.From = &tsBalance.timestamp
			params.To = &row.WindowStart
			params.WindowSize = nil
			params.WindowTimeZone = nil

			rows, err := e.queryMeter(ctx, owner.NamespacedID.Namespace, owner.Meter, params)
			if err != nil {
				return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("failed to query meter: %w", err)
			}

			var usage float64

			// We should have 1 row if there is usage data
			if len(rows) == 1 {
				usage = rows[0].Value
			} else if len(rows) > 1 {
				return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("expected at most 1 row, got %d", len(rows))
			}

			// deduct balance and increase overage if needed

			balanceAtEnd := math.Max(0, tsBalance.balance-usage)
			deductedUsage := tsBalance.balance - balanceAtEnd
			overage := usage - deductedUsage + tsBalance.overage

			// update
			tsBalance.balance = balanceAtEnd
			tsBalance.overage = overage
		}

		// Let's mark this balance as visited
		visited[idx] = true

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

	return windows, historyRes.History, nil
}

// queryMeter is a wrapper around streamingConnector.QueryMeter that accepts a 0 length period and returns 0 for it
func (e *connector) queryMeter(ctx context.Context, namespace string, m meter.Meter, params streaming.QueryParams) ([]meter.MeterQueryRow, error) {
	if params.From != nil && params.To != nil && params.From.Equal(*params.To) {
		return []meter.MeterQueryRow{
			{
				Value:       0,
				WindowStart: *params.From,
				WindowEnd:   *params.To,
			},
		}, nil
	}

	return e.streamingConnector.QueryMeter(ctx, namespace, m, params)
}
