// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package meteredentitlement

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
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
	nsOwner := grant.NamespacedOwner{
		Namespace: entitlementID.Namespace,
		ID:        grant.Owner(entitlementID.ID),
	}
	res, err := e.balanceConnector.GetBalanceOfOwner(ctx, nsOwner, at)
	if err != nil {
		if _, ok := err.(*grant.OwnerNotFoundError); ok {
			return nil, &entitlement.NotFoundError{EntitlementID: entitlementID}
		}
		return nil, err
	}

	ownerMeter, err := e.ownerConnector.GetMeter(ctx, nsOwner)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner query params: %w", err)
	}

	startOfPeriod, err := e.ownerConnector.GetUsagePeriodStartAt(ctx, nsOwner, at)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage period start at: %w", err)
	}

	meterQuery := ownerMeter.DefaultParams
	meterQuery.FilterSubject = []string{ownerMeter.SubjectKey}
	meterQuery.From = &startOfPeriod
	meterQuery.To = &at

	// We round up to closest full minute to include all the partial usage in the last minute of querying
	if trunc := meterQuery.To.Truncate(time.Minute); trunc.Before(*meterQuery.To) {
		meterQuery.To = convert.ToPointer(trunc.Add(time.Minute))
	}

	rows, err := e.streamingConnector.QueryMeter(ctx, entitlementID.Namespace, ownerMeter.MeterSlug, meterQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query meter: %w", err)
	}

	// TODO: refactor, assert 1 row
	usage := 0.0
	for _, row := range rows {
		usage += row.Value
	}

	return &EntitlementBalance{
		EntitlementID: entitlementID.ID,
		Balance:       res.Balance(),
		UsageInPeriod: usage,
		Overage:       res.Overage,
		StartOfPeriod: startOfPeriod,
	}, nil
}

func (e *connector) GetEntitlementBalanceHistory(ctx context.Context, entitlementID models.NamespacedID, params BalanceHistoryParams) ([]EntitlementBalanceHistoryWindow, engine.GrantBurnDownHistory, error) {
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
		return nil, engine.GrantBurnDownHistory{}, &models.GenericUserError{Message: fmt.Sprintf("from cannot be before %s", ent.MeasureUsageFrom.UTC().Format(time.RFC3339))}
	}

	owner := grant.NamespacedOwner{
		Namespace: entitlementID.Namespace,
		ID:        grant.Owner(entitlementID.ID),
	}

	ownerMeter, err := e.ownerConnector.GetMeter(ctx, owner)
	if err != nil {
		return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("failed to get owner query params: %w", err)
	}

	// 1. we get the burndown history
	burndownHistory, err := e.balanceConnector.GetBalanceHistoryOfOwner(ctx, owner, credit.BalanceHistoryParams{
		From: params.From.Truncate(ownerMeter.WindowSize.Duration()),
		To:   params.To.Truncate(ownerMeter.WindowSize.Duration()),
	})
	if err != nil {
		return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("failed to get balance history: %w", err)
	}

	// 2. and we get the windowed usage data
	meterQuery := ownerMeter.DefaultParams
	meterQuery.FilterSubject = []string{ownerMeter.SubjectKey}
	meterQuery.From = params.From
	meterQuery.To = params.To
	meterQuery.WindowSize = convert.ToPointer(models.WindowSize(params.WindowSize))
	meterQuery.WindowTimeZone = &params.WindowTimeZone

	meterRows, err := e.streamingConnector.QueryMeter(ctx, owner.Namespace, ownerMeter.MeterSlug, meterQuery)
	if err != nil {
		return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("failed to query meter: %w", err)
	}

	// If we get 0 rows that means the windowsize is larger than the queried period.
	// In this case we simply query for the entire period.
	if len(meterRows) == 0 {
		nonWindowedParams := *meterQuery
		nonWindowedParams.FilterSubject = []string{ownerMeter.SubjectKey}
		nonWindowedParams.WindowSize = nil
		nonWindowedParams.WindowTimeZone = nil
		meterRows, err = e.streamingConnector.QueryMeter(ctx, owner.Namespace, ownerMeter.MeterSlug, &nonWindowedParams)
		if err != nil {
			return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("failed to query meter: %w", err)
		}
	}

	// 3. and then we merge the two

	// convert history segments to list of point in time balances
	segments := burndownHistory.Segments()

	if len(segments) == 0 {
		return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("returned history is empty")
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
		},
		) bool {
			return tsb.timestamp.Before(row.WindowStart) || tsb.timestamp.Equal(row.WindowStart)
		}, true)
		if !ok {
			return nil, engine.GrantBurnDownHistory{}, fmt.Errorf("no balance found for time %s", row.WindowStart.Format(time.RFC3339))
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

	return windows, burndownHistory, nil
}
