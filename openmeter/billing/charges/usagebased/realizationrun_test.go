package usagebased

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
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
		priorRunID  *RealizationRunID
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
			priorRunID:  lo.ToPtr(RealizationRunID{Namespace: "namespace", ID: "run-2"}),
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
			priorRunID: lo.ToPtr(RealizationRunID{Namespace: "namespace", ID: "run-1"}),
			wantErr:    true,
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
			priorRunID:  lo.ToPtr(RealizationRunID{Namespace: "namespace", ID: "run-1"}),
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
			priorRunID:  lo.ToPtr(RealizationRunID{Namespace: "namespace", ID: "run-1"}),
			wantLine:    15,
			wantPreLine: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.currentRun.PriorRunID = tt.priorRunID

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

func TestRealizationRuns_MapToBillingMeteredQuantityUsesPriorRunLineage(t *testing.T) {
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	runs := RealizationRuns{
		newRealizationRunForBillingMeteredQuantityTest(
			"referenced-run",
			RealizationRunTypePartialInvoice,
			periodStart.Add(24*time.Hour),
			5,
		),
		newRealizationRunForBillingMeteredQuantityTest(
			"later-unreferenced-run",
			RealizationRunTypePartialInvoice,
			periodStart.Add(48*time.Hour),
			8,
		),
	}
	currentRun := newRealizationRunForBillingMeteredQuantityTest(
		"current",
		RealizationRunTypeFinalRealization,
		periodStart.Add(72*time.Hour),
		20,
	)
	currentRun.PriorRunID = lo.ToPtr(runs[0].ID)

	billingMeteredQuantity, err := runs.MapToBillingMeteredQuantity(currentRun)
	require.NoError(t, err)
	require.Equal(t, 15.0, billingMeteredQuantity.LinePeriod.InexactFloat64())
	require.Equal(t, 5.0, billingMeteredQuantity.PreLinePeriod.InexactFloat64())
}

func TestRealizationRuns_MapToBillingMeteredQuantityUsesZeroForFirstRun(t *testing.T) {
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	runs := RealizationRuns{
		newRealizationRunForBillingMeteredQuantityTest(
			"unrelated-run",
			RealizationRunTypePartialInvoice,
			periodStart.Add(24*time.Hour),
			5,
		),
	}
	currentRun := newRealizationRunForBillingMeteredQuantityTest(
		"current",
		RealizationRunTypeFinalRealization,
		periodStart.Add(48*time.Hour),
		20,
	)
	billingMeteredQuantity, err := runs.MapToBillingMeteredQuantity(currentRun)
	require.NoError(t, err)
	require.Equal(t, 20.0, billingMeteredQuantity.LinePeriod.InexactFloat64())
	require.Equal(t, 0.0, billingMeteredQuantity.PreLinePeriod.InexactFloat64())
}

func TestRealizationRuns_MapToBillingMeteredQuantityRejectsMissingPriorRun(t *testing.T) {
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	currentRun := newRealizationRunForBillingMeteredQuantityTest(
		"current",
		RealizationRunTypeFinalRealization,
		periodStart.Add(48*time.Hour),
		20,
	)
	currentRun.PriorRunID = lo.ToPtr(RealizationRunID{Namespace: "namespace", ID: "missing-run"})

	_, err := (RealizationRuns{}).MapToBillingMeteredQuantity(currentRun)
	require.ErrorContains(t, err, "resolve prior realization run missing-run")
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

func TestRealizationRuns_Latest(t *testing.T) {
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("empty", func(t *testing.T) {
		_, ok := RealizationRuns{}.Latest()

		require.False(t, ok)
	})

	t.Run("latest service period end", func(t *testing.T) {
		run, ok := RealizationRuns{
			newRealizationRunForBillingMeteredQuantityTest(
				"run-1",
				RealizationRunTypePartialInvoice,
				periodStart.Add(24*time.Hour),
				1,
			),
			newRealizationRunForBillingMeteredQuantityTest(
				"run-2",
				RealizationRunTypePartialInvoice,
				periodStart.Add(48*time.Hour),
				1,
			),
		}.Latest()

		require.True(t, ok)
		require.Equal(t, "run-2", run.ID.ID)
	})

	t.Run("latest created at wins same service period end", func(t *testing.T) {
		periodEnd := periodStart.Add(24 * time.Hour)
		older := newRealizationRunForBillingMeteredQuantityTest(
			"older",
			RealizationRunTypePartialInvoice,
			periodEnd,
			1,
		)
		newer := newRealizationRunForBillingMeteredQuantityTest(
			"newer",
			RealizationRunTypePartialInvoice,
			periodEnd,
			1,
		)
		newer.CreatedAt = older.CreatedAt.Add(time.Hour)

		run, ok := RealizationRuns{newer, older}.Latest()

		require.True(t, ok)
		require.Equal(t, "newer", run.ID.ID)
	})
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

func TestCharge_BisectRealizationRunsByTimestamp(t *testing.T) {
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
	charge := newChargeWithRealizationsForServicePeriodTest(servicePeriod, runs)

	before, containingOrAfter, err := charge.BisectRealizationRunsByTimestamp(at)
	require.NoError(t, err)

	require.Equal(t, []string{"before-run"}, lo.Map(before, func(run RealizationRun, _ int) string {
		return run.ID.ID
	}))
	require.Equal(t, []string{"containing-run", "after-run"}, lo.Map(containingOrAfter, func(run RealizationRun, _ int) string {
		return run.ID.ID
	}))

	before, containingOrAfter, err = charge.BisectRealizationRunsByTimestamp(periodStart.Add(48 * time.Hour))
	require.NoError(t, err)

	require.Equal(t, []string{"before-run", "containing-run"}, lo.Map(before, func(run RealizationRun, _ int) string {
		return run.ID.ID
	}))
	require.Equal(t, []string{"after-run"}, lo.Map(containingOrAfter, func(run RealizationRun, _ int) string {
		return run.ID.ID
	}))
}

func TestCharge_ServicePeriodForUsesPriorRunLineage(t *testing.T) {
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	intentPeriod := timeutil.ClosedPeriod{
		From: periodStart,
		To:   periodStart.Add(96 * time.Hour),
	}
	priorRun := newRealizationRunForBillingMeteredQuantityTest(
		"prior-run",
		RealizationRunTypePartialInvoice,
		periodStart.Add(24*time.Hour),
		5,
	)
	currentRun := newRealizationRunForBillingMeteredQuantityTest(
		"current-run",
		RealizationRunTypeFinalRealization,
		periodStart.Add(72*time.Hour),
		20,
	)
	currentRun.PriorRunID = lo.ToPtr(priorRun.ID)
	charge := newChargeWithRealizationsForServicePeriodTest(intentPeriod, RealizationRuns{priorRun, currentRun})

	period, err := charge.ServicePeriodFor(currentRun)
	require.NoError(t, err)
	require.Equal(t, priorRun.ServicePeriodTo, period.From)
	require.Equal(t, currentRun.ServicePeriodTo, period.To)
}

func TestCharge_ServicePeriodForFirstRunUsesIntentStart(t *testing.T) {
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	intentPeriod := timeutil.ClosedPeriod{
		From: periodStart,
		To:   periodStart.Add(48 * time.Hour),
	}
	currentRun := newRealizationRunForBillingMeteredQuantityTest(
		"current-run",
		RealizationRunTypeFinalRealization,
		periodStart.Add(24*time.Hour),
		20,
	)
	charge := newChargeWithRealizationsForServicePeriodTest(intentPeriod, RealizationRuns{currentRun})

	period, err := charge.ServicePeriodFor(currentRun)
	require.NoError(t, err)
	require.Equal(t, intentPeriod.From, period.From)
	require.Equal(t, currentRun.ServicePeriodTo, period.To)
}

func newChargeWithRealizationsForServicePeriodTest(servicePeriod timeutil.ClosedPeriod, runs RealizationRuns) Charge {
	return Charge{
		ChargeBase: ChargeBase{
			Intent: Intent{
				IntentMutableFields: IntentMutableFields{
					IntentMutableFields: chargesmeta.IntentMutableFields{
						ServicePeriod: servicePeriod,
					},
				},
			}.AsOverridableIntent(),
		},
		Realizations: runs,
	}
}

func TestRealizationRuns_PriorRunIDForNextRunUsesCreatedOrderAndSkipsVoidedHistory(t *testing.T) {
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deletedAt := periodStart.Add(time.Hour)

	validRun := newRealizationRunForBillingMeteredQuantityTest(
		"valid-run",
		RealizationRunTypePartialInvoice,
		periodStart.Add(24*time.Hour),
		5,
	)
	newerValidRun := newRealizationRunForBillingMeteredQuantityTest(
		"newer-valid-run",
		RealizationRunTypePartialInvoice,
		periodStart.Add(12*time.Hour),
		3,
	)
	newerValidRun.CreatedAt = periodStart.Add(96 * time.Hour)
	invalidRun := newRealizationRunForBillingMeteredQuantityTest(
		"invalid-run",
		RealizationRunTypeInvalidDueToUnsupportedCreditNote,
		periodStart.Add(48*time.Hour),
		8,
	)
	deletedRun := newRealizationRunForBillingMeteredQuantityTest(
		"deleted-run",
		RealizationRunTypePartialInvoice,
		periodStart.Add(72*time.Hour),
		10,
	)
	deletedRun.DeletedAt = &deletedAt

	priorRunID := (RealizationRuns{validRun, newerValidRun, invalidRun, deletedRun}).PriorRunIDForNextRun()
	require.Equal(t, &newerValidRun.ID, priorRunID)
	require.Nil(t, (RealizationRuns{}).PriorRunIDForNextRun())
}

func TestRealizationRuns_PriorRunIDForNextRunUsesIDToBreakCreatedAtTie(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	firstRun := newRealizationRunForBillingMeteredQuantityTest(
		"01-first-run",
		RealizationRunTypePartialInvoice,
		createdAt.Add(24*time.Hour),
		5,
	)
	firstRun.CreatedAt = createdAt
	secondRun := newRealizationRunForBillingMeteredQuantityTest(
		"02-second-run",
		RealizationRunTypePartialInvoice,
		createdAt.Add(48*time.Hour),
		10,
	)
	secondRun.CreatedAt = createdAt

	priorRunID := (RealizationRuns{secondRun, firstRun}).PriorRunIDForNextRun()
	require.Equal(t, &secondRun.ID, priorRunID)
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
