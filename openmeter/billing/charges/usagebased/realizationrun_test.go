package usagebased

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
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

func TestRealizationRuns_GetByLineID(t *testing.T) {
	lineID := "line-1"
	otherLineID := "line-2"

	runs := RealizationRuns{
		{
			RealizationRunBase: RealizationRunBase{
				ID:     RealizationRunID{Namespace: "namespace", ID: "run-without-line"},
				LineID: nil,
			},
		},
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

func newRealizationRunForBillingMeteredQuantityTest(id string, typ RealizationRunType, servicePeriodTo time.Time, meteredQuantity int64) RealizationRun {
	return RealizationRun{
		RealizationRunBase: RealizationRunBase{
			ID: RealizationRunID{
				Namespace: "namespace",
				ID:        id,
			},
			Type:            typ,
			ServicePeriodTo: servicePeriodTo,
			MeteredQuantity: alpacadecimal.NewFromInt(meteredQuantity),
		},
	}
}
