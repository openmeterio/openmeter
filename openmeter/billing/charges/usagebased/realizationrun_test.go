package usagebased

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestRealizationRuns_MapToBillingMeteredQuantity(t *testing.T) {
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		runs        RealizationRuns
		currentRun  RealizationRun
		wantLine    float64
		wantPreLine float64
		wantErr     bool
	}{
		{
			name: "first run has no pre-line period quantity",
			currentRun: newRealizationRunForBillingMeteredQuantityTest(
				"current",
				RealizationRunTypePartialInvoice,
				periodStart.Add(24*time.Hour),
				5,
			),
			wantLine:    5,
			wantPreLine: 0,
		},
		{
			name: "uses latest prior persisted cumulative quantity",
			runs: RealizationRuns{
				newRealizationRunForBillingMeteredQuantityTest(
					"run-1",
					RealizationRunTypePartialInvoice,
					periodStart.Add(24*time.Hour),
					5,
				),
				newRealizationRunForBillingMeteredQuantityTest(
					"run-2",
					RealizationRunTypePartialInvoice,
					periodStart.Add(48*time.Hour),
					8,
				),
			},
			currentRun: newRealizationRunForBillingMeteredQuantityTest(
				"current",
				RealizationRunTypeFinalRealization,
				periodStart.Add(72*time.Hour),
				20,
			),
			wantLine:    12,
			wantPreLine: 8,
		},
		{
			name: "errors when current cumulative quantity is below prior billed quantity",
			runs: RealizationRuns{
				newRealizationRunForBillingMeteredQuantityTest(
					"run-1",
					RealizationRunTypePartialInvoice,
					periodStart.Add(24*time.Hour),
					10,
				),
			},
			currentRun: newRealizationRunForBillingMeteredQuantityTest(
				"current",
				RealizationRunTypeFinalRealization,
				periodStart.Add(48*time.Hour),
				5,
			),
			wantErr: true,
		},
		{
			name: "ignores deleted prior runs",
			runs: RealizationRuns{
				func() RealizationRun {
					run := newRealizationRunForBillingMeteredQuantityTest(
						"deleted-run",
						RealizationRunTypePartialInvoice,
						periodStart.Add(48*time.Hour),
						18,
					)
					run.DeletedAt = &periodStart
					return run
				}(),
				newRealizationRunForBillingMeteredQuantityTest(
					"run-1",
					RealizationRunTypePartialInvoice,
					periodStart.Add(24*time.Hour),
					5,
				),
			},
			currentRun: newRealizationRunForBillingMeteredQuantityTest(
				"current",
				RealizationRunTypeFinalRealization,
				periodStart.Add(72*time.Hour),
				20,
			),
			wantLine:    15,
			wantPreLine: 5,
		},
		{
			name: "ignores invalid unsupported credit note prior runs",
			runs: RealizationRuns{
				newRealizationRunForBillingMeteredQuantityTest(
					"run-1",
					RealizationRunTypePartialInvoice,
					periodStart.Add(24*time.Hour),
					5,
				),
				newRealizationRunForBillingMeteredQuantityTest(
					"invalid-run",
					RealizationRunTypeInvalidDueToUnsupportedCreditNote,
					periodStart.Add(48*time.Hour),
					18,
				),
			},
			currentRun: newRealizationRunForBillingMeteredQuantityTest(
				"current",
				RealizationRunTypeFinalRealization,
				periodStart.Add(72*time.Hour),
				20,
			),
			wantLine:    15,
			wantPreLine: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			billingMeteredQuantity, err := tt.runs.MapToBillingMeteredQuantity(tt.currentRun)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantLine, billingMeteredQuantity.LinePeriod.InexactFloat64())
			require.Equal(t, tt.wantPreLine, billingMeteredQuantity.PreLinePeriod.InexactFloat64())
		})
	}
}

func TestRealizationRunType_IsVoidedBillingHistory(t *testing.T) {
	require.False(t, RealizationRunTypeFinalRealization.IsVoidedBillingHistory())
	require.False(t, RealizationRunTypePartialInvoice.IsVoidedBillingHistory())
	require.True(t, RealizationRunTypeInvalidDueToUnsupportedCreditNote.IsVoidedBillingHistory())
}

func TestRealizationRun_InvalidUnsupportedCreditNoteKeepsInitialType(t *testing.T) {
	run := newRealizationRunForBillingMeteredQuantityTest(
		"invalid-run",
		RealizationRunTypeFinalRealization,
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		10,
	)
	run.Type = RealizationRunTypeInvalidDueToUnsupportedCreditNote

	require.NoError(t, run.Validate())
	require.Equal(t, RealizationRunTypeFinalRealization, run.InitialType)
	require.True(t, run.IsVoidedBillingHistory())
}

func TestRealizationRuns_SumSkipsVoidedBillingHistory(t *testing.T) {
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deletedAt := periodStart.Add(time.Hour)

	deletedRun := newRealizationRunForBillingMeteredQuantityTest(
		"deleted-run",
		RealizationRunTypePartialInvoice,
		periodStart.Add(24*time.Hour),
		100,
	)
	deletedRun.DeletedAt = &deletedAt

	invalidRun := newRealizationRunForBillingMeteredQuantityTest(
		"invalid-run",
		RealizationRunTypeInvalidDueToUnsupportedCreditNote,
		periodStart.Add(48*time.Hour),
		100,
	)

	effectiveRun := newRealizationRunForBillingMeteredQuantityTest(
		"effective-run",
		RealizationRunTypeFinalRealization,
		periodStart.Add(72*time.Hour),
		7,
	)

	require.Equal(t, float64(7), RealizationRuns{deletedRun, invalidRun, effectiveRun}.Sum().Total.InexactFloat64())
}

func TestRealizationRuns_GetByLineID(t *testing.T) {
	lineID := "line-1"
	otherLineID := "line-2"

	runs := RealizationRuns{
		{
			RealizationRunBase: RealizationRunBase{
				ID:     RealizationRunID{Namespace: "namespace", ID: "run-1"},
				LineID: &otherLineID,
			},
		},
		{
			RealizationRunBase: RealizationRunBase{
				ID:     RealizationRunID{Namespace: "namespace", ID: "run-2"},
				LineID: &lineID,
			},
		},
	}

	run, err := runs.GetByLineID(lineID)
	require.NoError(t, err)
	require.Equal(t, "run-2", run.ID.ID)

	_, err = runs.GetByLineID("missing-line")
	require.ErrorContains(t, err, "realization run not found")
}

func TestRealizationRuns_BisectByTimestamp(t *testing.T) {
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	servicePeriod := timeutil.ClosedPeriod{
		From: periodStart,
		To:   periodStart.Add(96 * time.Hour),
	}
	at := periodStart.Add(36 * time.Hour)

	deletedAt := periodStart.Add(time.Hour)
	runs := RealizationRuns{
		func() RealizationRun {
			run := newRealizationRunForBillingMeteredQuantityTest(
				"deleted-run",
				RealizationRunTypePartialInvoice,
				periodStart.Add(96*time.Hour),
				1,
			)
			run.DeletedAt = &deletedAt
			return run
		}(),
		newRealizationRunForBillingMeteredQuantityTest(
			"after-run",
			RealizationRunTypePartialInvoice,
			periodStart.Add(72*time.Hour),
			1,
		),
		newRealizationRunForBillingMeteredQuantityTest(
			"before-run",
			RealizationRunTypePartialInvoice,
			periodStart.Add(24*time.Hour),
			1,
		),
		newRealizationRunForBillingMeteredQuantityTest(
			"containing-run",
			RealizationRunTypePartialInvoice,
			periodStart.Add(48*time.Hour),
			1,
		),
	}

	before, containingOrAfter := runs.BisectByTimestamp(servicePeriod, at)

	require.Equal(t, []string{"before-run"}, lo.Map(before, func(run RealizationRun, _ int) string {
		return run.ID.ID
	}))
	require.Equal(t, []string{"containing-run", "after-run"}, lo.Map(containingOrAfter, func(run RealizationRun, _ int) string {
		return run.ID.ID
	}))

	before, containingOrAfter = runs.BisectByTimestamp(servicePeriod, periodStart.Add(48*time.Hour))

	require.Equal(t, []string{"before-run", "containing-run"}, lo.Map(before, func(run RealizationRun, _ int) string {
		return run.ID.ID
	}))
	require.Equal(t, []string{"after-run"}, lo.Map(containingOrAfter, func(run RealizationRun, _ int) string {
		return run.ID.ID
	}))
}

func newRealizationRunForBillingMeteredQuantityTest(id string, typ RealizationRunType, servicePeriodTo time.Time, meteredQuantity int64) RealizationRun {
	return RealizationRun{
		RealizationRunBase: RealizationRunBase{
			ID: RealizationRunID{
				Namespace: "namespace",
				ID:        id,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: servicePeriodTo.Add(-time.Hour),
				UpdatedAt: servicePeriodTo.Add(-time.Hour),
			},
			FeatureID:       "feature-1",
			Type:            typ,
			InitialType:     typ,
			StoredAtLT:      servicePeriodTo,
			ServicePeriodTo: servicePeriodTo,
			MeteredQuantity: alpacadecimal.NewFromInt(meteredQuantity),
			Totals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(meteredQuantity),
				Total:  alpacadecimal.NewFromInt(meteredQuantity),
			},
		},
	}
}
