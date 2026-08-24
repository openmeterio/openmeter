package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// newFlatFeeCharge builds a minimal flat-fee charge whose effective intent's
// service period is servicePeriod, matching the boilerplate api_attach_test.go
// uses for the usage-based charge. resolveFlatFeeRealizations neither
// validates nor reads any other field, so the rest is left at its zero value.
func newFlatFeeCharge(t *testing.T, status flatfee.Status, servicePeriod timeutil.ClosedPeriod, realizations flatfee.Realizations) flatfee.Charge {
	t.Helper()

	return flatfee.Charge{
		ChargeBase: flatfee.ChargeBase{
			ManagedResource: meta.ManagedResource{ID: "charge-1"},
			Status:          status,
			Intent: flatfee.NewOverridableIntent(flatfee.Intent{
				Intent: meta.Intent{
					CustomerID: "cust-1",
					Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "flat fee test charge",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt: servicePeriod.To,
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			}, nil),
		},
		Realizations: realizations,
	}
}

// newUsageBasedCharge builds a minimal usage-based charge whose effective
// intent's service period is servicePeriod, mirroring the boilerplate
// api_attach_test.go uses. resolveUsageBasedRealizations neither validates
// nor reads any other field, so the rest is left at its zero value.
func newUsageBasedCharge(t *testing.T, status usagebased.Status, servicePeriod timeutil.ClosedPeriod, runs usagebased.RealizationRuns, realtimeQuantity *alpacadecimal.Decimal) usagebased.Charge {
	t.Helper()

	return usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{ID: "charge-1"},
			Status:          status,
			Intent: usagebased.NewOverridableIntent(usagebased.Intent{
				Intent: meta.Intent{
					CustomerID: "cust-1",
					Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
				},
				IntentMutableFields: usagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "usage based test charge",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt: servicePeriod.To,
					Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			}, nil),
			State: usagebased.State{
				FeatureID:    "feature-1",
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
		Realizations: runs,
		Expands:      usagebased.Expands{RealtimeQuantity: realtimeQuantity},
	}
}

// TestResolveFlatFeeRealizations ports the deleted flatfee.Realizations.Resolve
// tests (TestRealizations_Resolve in the original flatfee package) onto the
// service-level resolveFlatFeeRealizations function.
func TestResolveFlatFeeRealizations(t *testing.T) {
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	servicePeriod := timeutil.ClosedPeriod{
		From: periodStart,
		To:   periodStart.Add(96 * time.Hour),
	}

	newRun := func(id string, typ flatfee.RealizationRunType, from, to time.Time, createdAt time.Time) flatfee.RealizationRun {
		return flatfee.RealizationRun{
			RealizationRunBase: flatfee.RealizationRunBase{
				ID: flatfee.RealizationRunID{Namespace: "namespace", ID: id},
				ManagedModel: models.ManagedModel{
					CreatedAt: createdAt,
					UpdatedAt: createdAt,
				},
				Type:          typ,
				InitialType:   flatfee.RealizationRunTypeFinalRealization,
				ServicePeriod: timeutil.ClosedPeriod{From: from, To: to},
			},
		}
	}

	// given a live prior run, a run voided by an unsupported credit note, and a
	// current run re-covering the voided period, supplied out of order
	priorRun := newRun("prior-run", flatfee.RealizationRunTypeFinalRealization, periodStart, periodStart.Add(24*time.Hour), periodStart)
	voidedRun := newRun("voided-run", flatfee.RealizationRunTypeInvalidDueToUnsupportedCreditNote, periodStart.Add(24*time.Hour), periodStart.Add(48*time.Hour), periodStart.Add(time.Hour))
	currentRun := newRun("current-run", flatfee.RealizationRunTypeFinalRealization, periodStart.Add(24*time.Hour), periodStart.Add(72*time.Hour), periodStart.Add(2*time.Hour))

	realizations := flatfee.Realizations{
		CurrentRun: &currentRun,
		PriorRuns:  flatfee.RealizationRuns{voidedRun, priorRun},
	}

	t.Run("returns the booked history when a live run exists", func(t *testing.T) {
		// given live and voided history; flat fees are never partially
		// invoiced, so any live run recognizes the whole period regardless of
		// the runs' own persisted periods
		charge := newFlatFeeCharge(t, flatfee.StatusActive, servicePeriod, realizations)

		resolved, err := resolveFlatFeeRealizations(charge, map[string]billing.StandardInvoice{})
		require.NoError(t, err)

		// then the booked history is returned in creation order, voided runs
		// marked as inert audit entries, and no outstanding projection is
		// emitted
		require.Len(t, resolved, 3)

		require.Equal(t, "prior-run", resolved[0].Run.ID.ID)
		require.Equal(t, priorRun.ServicePeriod, resolved[0].ServicePeriod)
		require.False(t, resolved[0].Voided)

		require.Equal(t, "voided-run", resolved[1].Run.ID.ID)
		require.Equal(t, voidedRun.ServicePeriod, resolved[1].ServicePeriod)
		require.True(t, resolved[1].Voided)

		require.Equal(t, "current-run", resolved[2].Run.ID.ID)
		require.False(t, resolved[2].Voided)
	})

	t.Run("no outstanding projection for final and deleted charges", func(t *testing.T) {
		for _, status := range []flatfee.Status{flatfee.StatusFinal, flatfee.StatusDeleted} {
			charge := newFlatFeeCharge(t, status, servicePeriod, realizations)

			resolved, err := resolveFlatFeeRealizations(charge, map[string]billing.StandardInvoice{})
			require.NoError(t, err)
			require.Len(t, resolved, 3)
			require.NotNil(t, resolved[2].Run)
		}
	})

	t.Run("deleted runs do not cover the period", func(t *testing.T) {
		// given only a deleted (voided) run; voided history does not recognize
		// the period, so the charge is entirely outstanding and the voided
		// entry is dropped in favor of the projection
		deletedAt := periodStart.Add(time.Hour)
		deletedRun := newRun("deleted-run", flatfee.RealizationRunTypeFinalRealization, periodStart, periodStart.Add(48*time.Hour), periodStart)
		deletedRun.DeletedAt = &deletedAt

		charge := newFlatFeeCharge(t, flatfee.StatusActive, servicePeriod, flatfee.Realizations{PriorRuns: flatfee.RealizationRuns{deletedRun}})

		resolved, err := resolveFlatFeeRealizations(charge, map[string]billing.StandardInvoice{})
		require.NoError(t, err)

		require.Len(t, resolved, 1)
		require.Nil(t, resolved[0].Run)
		require.Equal(t, servicePeriod, resolved[0].ServicePeriod)
	})

	t.Run("attaches invoice lines by line id", func(t *testing.T) {
		// given a booked run with a line reference, and one with an invoice
		// reference but no line (a legal domain state: the two fields validate
		// independently)
		linedRun := newRun("lined-run", flatfee.RealizationRunTypeFinalRealization, periodStart, periodStart.Add(48*time.Hour), periodStart)
		linedRun.LineID = lo.ToPtr("line-1")
		linedRun.InvoiceID = lo.ToPtr("inv-1")

		linelessRun := newRun("lineless-run", flatfee.RealizationRunTypeFinalRealization, periodStart.Add(48*time.Hour), servicePeriod.To, periodStart.Add(time.Hour))
		linelessRun.InvoiceID = lo.ToPtr("inv-1")

		charge := newFlatFeeCharge(t, flatfee.StatusActive, servicePeriod, flatfee.Realizations{PriorRuns: flatfee.RealizationRuns{linedRun, linelessRun}})

		lines := map[string]billing.StandardInvoice{
			"line-1": {StandardInvoiceBase: billing.StandardInvoiceBase{ID: "inv-1"}},
		}

		resolved, err := resolveFlatFeeRealizations(charge, lines)
		require.NoError(t, err)

		// then only the run booked to a known line carries the attached
		// invoice; the line-less run stays bare
		require.Len(t, resolved, 2)
		require.NotNil(t, resolved[0].Invoice, "the lined run carries the attached invoice")
		require.Equal(t, "inv-1", resolved[0].Invoice.ID)
		require.Nil(t, resolved[1].Invoice, "a run without a line id cannot attach an invoice")
	})
}

func newUsageBasedRealizationRun(id string, typ usagebased.RealizationRunType, servicePeriodTo time.Time, meteredQuantity int64) usagebased.RealizationRun {
	return usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
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

// TestResolveUsageBasedRealizations ports the deleted
// usagebased.RealizationRuns.Resolve and
// usagebased.ResolvedRealizations.ApplyRealtimeQuantity tests (respectively
// TestRealizationRuns_Resolve and TestResolvedRealizations_ApplyRealtimeQuantity
// in the original usagebased package) onto the service-level
// resolveUsageBasedRealizations function, which now performs both steps in one
// call. It also adds a regression test pinning that signed booked deltas
// telescope into the outstanding remainder, so the realization list
// reconciles with the live read after a downward correction.
func TestResolveUsageBasedRealizations(t *testing.T) {
	periodStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	servicePeriod := timeutil.ClosedPeriod{
		From: periodStart,
		To:   periodStart.Add(96 * time.Hour),
	}

	deletedAt := periodStart.Add(time.Hour)

	newDeletedRun := func(id string, servicePeriodTo time.Time, meteredQuantity int64) usagebased.RealizationRun {
		run := newUsageBasedRealizationRun(id, usagebased.RealizationRunTypePartialInvoice, servicePeriodTo, meteredQuantity)
		run.DeletedAt = &deletedAt
		return run
	}

	t.Run("orders, stitches and de-cumulates around voided history", func(t *testing.T) {
		// given two live runs supplied out of order plus a deleted one
		runs := usagebased.RealizationRuns{
			newUsageBasedRealizationRun("run-2", usagebased.RealizationRunTypePartialInvoice, periodStart.Add(48*time.Hour), 25),
			newDeletedRun("deleted-run", periodStart.Add(72*time.Hour), 40),
			newUsageBasedRealizationRun("run-1", usagebased.RealizationRunTypePartialInvoice, periodStart.Add(24*time.Hour), 10),
		}
		charge := newUsageBasedCharge(t, usagebased.StatusActive, servicePeriod, runs, nil)

		// when resolving the presentation view
		resolved, err := resolveUsageBasedRealizations(charge, map[string]billing.StandardInvoice{})
		require.NoError(t, err)

		// then live runs are stitched into contiguous periods with per-run
		// quantities, the voided run is an inert audit entry, and the tail not
		// covered by live runs is projected as outstanding
		require.Len(t, resolved, 4)

		require.Equal(t, "run-1", resolved[0].Run.ID.ID)
		require.Equal(t, timeutil.ClosedPeriod{From: periodStart, To: periodStart.Add(24 * time.Hour)}, resolved[0].ServicePeriod)
		require.Equal(t, float64(10), resolved[0].Quantity.InexactFloat64())
		require.False(t, resolved[0].Voided)

		require.Equal(t, "run-2", resolved[1].Run.ID.ID)
		require.Equal(t, timeutil.ClosedPeriod{From: periodStart.Add(24 * time.Hour), To: periodStart.Add(48 * time.Hour)}, resolved[1].ServicePeriod)
		require.Equal(t, float64(15), resolved[1].Quantity.InexactFloat64(), "quantity is the run's own consumption, not the cumulative one")
		require.False(t, resolved[1].Voided)

		require.Equal(t, "deleted-run", resolved[2].Run.ID.ID)
		require.Equal(t, timeutil.ClosedPeriod{From: periodStart.Add(48 * time.Hour), To: periodStart.Add(72 * time.Hour)}, resolved[2].ServicePeriod)
		require.Equal(t, float64(15), resolved[2].Quantity.InexactFloat64())
		require.True(t, resolved[2].Voided)

		require.Nil(t, resolved[3].Run, "the outstanding projection is not a persisted run")
		require.Equal(t, timeutil.ClosedPeriod{From: periodStart.Add(48 * time.Hour), To: servicePeriod.To}, resolved[3].ServicePeriod,
			"outstanding starts at the last live run's end; voided history does not cover the period")
		require.Equal(t, float64(0), resolved[3].Quantity.InexactFloat64())
		require.False(t, resolved[3].Voided)
	})

	t.Run("outstanding starts at the furthest non-voided period end", func(t *testing.T) {
		// given two live runs whose creation order inverts their period ends:
		// the later-created run books an earlier service-period end. The
		// outstanding projection must start at the furthest non-voided end
		// (Max over non-voided ServicePeriodTo), not at the last-created
		// run's end, or it would re-open already-recognized time.
		early := newUsageBasedRealizationRun("run-early", usagebased.RealizationRunTypePartialInvoice, periodStart.Add(72*time.Hour), 30)
		late := newUsageBasedRealizationRun("run-late", usagebased.RealizationRunTypePartialInvoice, periodStart.Add(48*time.Hour), 25)
		late.CreatedAt = periodStart.Add(73 * time.Hour)
		late.UpdatedAt = late.CreatedAt

		charge := newUsageBasedCharge(t, usagebased.StatusActive, servicePeriod, usagebased.RealizationRuns{late, early}, nil)

		resolved, err := resolveUsageBasedRealizations(charge, map[string]billing.StandardInvoice{})
		require.NoError(t, err)

		require.Len(t, resolved, 3)
		require.Equal(t, "run-early", resolved[0].Run.ID.ID)
		require.Equal(t, "run-late", resolved[1].Run.ID.ID)

		require.Nil(t, resolved[2].Run, "the outstanding projection is not a persisted run")
		require.Equal(t, timeutil.ClosedPeriod{From: periodStart.Add(72 * time.Hour), To: servicePeriod.To}, resolved[2].ServicePeriod,
			"outstanding starts at the furthest non-voided period end, not the last-created run's end")
	})

	t.Run("no outstanding projection for final and deleted charges", func(t *testing.T) {
		runs := usagebased.RealizationRuns{
			newUsageBasedRealizationRun("run-1", usagebased.RealizationRunTypeFinalRealization, periodStart.Add(24*time.Hour), 10),
		}

		for _, status := range []usagebased.Status{usagebased.StatusFinal, usagebased.StatusDeleted} {
			charge := newUsageBasedCharge(t, status, servicePeriod, runs, nil)

			resolved, err := resolveUsageBasedRealizations(charge, map[string]billing.StandardInvoice{})
			require.NoError(t, err)
			require.Len(t, resolved, 1)
			require.NotNil(t, resolved[0].Run)
		}
	})

	t.Run("no outstanding projection once live runs cover the service period", func(t *testing.T) {
		runs := usagebased.RealizationRuns{
			newUsageBasedRealizationRun("run-1", usagebased.RealizationRunTypeFinalRealization, servicePeriod.To, 10),
		}
		charge := newUsageBasedCharge(t, usagebased.StatusActive, servicePeriod, runs, nil)

		resolved, err := resolveUsageBasedRealizations(charge, map[string]billing.StandardInvoice{})
		require.NoError(t, err)
		require.Len(t, resolved, 1)
	})

	t.Run("emits a live run's shrunken cumulative as a signed delta", func(t *testing.T) {
		// given a downward usage correction: run-2's cumulative snapshot is
		// smaller than run-1's
		runs := usagebased.RealizationRuns{
			newUsageBasedRealizationRun("run-1", usagebased.RealizationRunTypePartialInvoice, periodStart.Add(24*time.Hour), 10),
			newUsageBasedRealizationRun("run-2", usagebased.RealizationRunTypeFinalRealization, periodStart.Add(48*time.Hour), 5),
		}
		charge := newUsageBasedCharge(t, usagebased.StatusActive, servicePeriod, runs, nil)

		// then the correction is reported frankly as a negative quantity
		// instead of a fabricated zero
		resolved, err := resolveUsageBasedRealizations(charge, map[string]billing.StandardInvoice{})
		require.NoError(t, err)
		require.Len(t, resolved, 3)
		require.Equal(t, float64(10), resolved[0].Quantity.InexactFloat64())
		require.Equal(t, float64(-5), resolved[1].Quantity.InexactFloat64())
		require.False(t, resolved[1].Voided)
	})

	t.Run("emits a voided run's negative delta signed", func(t *testing.T) {
		// given a voided run whose quantity snapshot predates the live run's
		runs := usagebased.RealizationRuns{
			newUsageBasedRealizationRun("run-1", usagebased.RealizationRunTypePartialInvoice, periodStart.Add(24*time.Hour), 10),
			newDeletedRun("deleted-run", periodStart.Add(48*time.Hour), 4),
		}
		charge := newUsageBasedCharge(t, usagebased.StatusActive, servicePeriod, runs, nil)

		resolved, err := resolveUsageBasedRealizations(charge, map[string]billing.StandardInvoice{})
		require.NoError(t, err)

		require.Len(t, resolved, 3)
		require.True(t, resolved[1].Voided)
		require.Equal(t, float64(-6), resolved[1].Quantity.InexactFloat64())
	})

	t.Run("outstanding gets the not-yet-booked remainder of the live read", func(t *testing.T) {
		// given a booked run (cumulative 10), a voided run, and an outstanding
		// tail, with a live cumulative read of 14
		deletedRun := newUsageBasedRealizationRun("deleted-run", usagebased.RealizationRunTypePartialInvoice, periodStart.Add(48*time.Hour), 12)
		deletedRun.DeletedAt = &deletedAt

		runs := usagebased.RealizationRuns{
			newUsageBasedRealizationRun("run-1", usagebased.RealizationRunTypePartialInvoice, periodStart.Add(24*time.Hour), 10),
			deletedRun,
		}
		realtimeQuantity := alpacadecimal.NewFromInt(14)
		charge := newUsageBasedCharge(t, usagebased.StatusActive, servicePeriod, runs, &realtimeQuantity)

		// when resolving with the live read supplied through the realtime
		// usage expand
		resolved, err := resolveUsageBasedRealizations(charge, map[string]billing.StandardInvoice{})
		require.NoError(t, err)
		require.Len(t, resolved, 3)

		// then only the outstanding projection changes: live read minus the
		// booked non-voided cumulative, ignoring voided audit entries
		require.Equal(t, float64(10), resolved[0].Quantity.InexactFloat64())
		require.True(t, resolved[1].Voided)
		require.Nil(t, resolved[2].Run)
		require.Equal(t, float64(4), resolved[2].Quantity.InexactFloat64())
	})

	t.Run("clamps to zero when booked history is ahead of the live read", func(t *testing.T) {
		runs := usagebased.RealizationRuns{
			newUsageBasedRealizationRun("run-1", usagebased.RealizationRunTypePartialInvoice, periodStart.Add(24*time.Hour), 10),
		}
		realtimeQuantity := alpacadecimal.NewFromInt(7)
		charge := newUsageBasedCharge(t, usagebased.StatusActive, servicePeriod, runs, &realtimeQuantity)

		resolved, err := resolveUsageBasedRealizations(charge, map[string]billing.StandardInvoice{})
		require.NoError(t, err)

		require.Nil(t, resolved[1].Run)
		require.Equal(t, float64(0), resolved[1].Quantity.InexactFloat64())
	})

	t.Run("no-op without an outstanding projection", func(t *testing.T) {
		runs := usagebased.RealizationRuns{
			newUsageBasedRealizationRun("run-1", usagebased.RealizationRunTypeFinalRealization, servicePeriod.To, 10),
		}
		realtimeQuantity := alpacadecimal.NewFromInt(14)
		charge := newUsageBasedCharge(t, usagebased.StatusActive, servicePeriod, runs, &realtimeQuantity)

		resolved, err := resolveUsageBasedRealizations(charge, map[string]billing.StandardInvoice{})
		require.NoError(t, err)
		require.Len(t, resolved, 1)

		require.Equal(t, float64(10), resolved[0].Quantity.InexactFloat64(), "booked quantities are billing facts and stay untouched")
	})

	t.Run("signed booked deltas telescope into the outstanding remainder", func(t *testing.T) {
		// given two live runs with cumulative MeteredQuantity 10 then 4 (a
		// legitimate downward correction): the signed per-entry deltas are 10
		// and -6, so the booked total telescopes to the last cumulative (4)
		runs := usagebased.RealizationRuns{
			newUsageBasedRealizationRun("run-1", usagebased.RealizationRunTypePartialInvoice, periodStart.Add(24*time.Hour), 10),
			newUsageBasedRealizationRun("run-2", usagebased.RealizationRunTypeFinalRealization, periodStart.Add(48*time.Hour), 4),
		}
		realtimeQuantity := alpacadecimal.NewFromInt(6)
		charge := newUsageBasedCharge(t, usagebased.StatusActive, servicePeriod, runs, &realtimeQuantity)

		// when resolving with a live read of 6 and the service period still
		// open past the last run
		resolved, err := resolveUsageBasedRealizations(charge, map[string]billing.StandardInvoice{})
		require.NoError(t, err)
		require.Len(t, resolved, 3)

		// then the booked entries state the correction frankly (10, -6) and
		// the outstanding entry reports the live read minus the booked
		// cumulative: 6 - 4 = 2. The entries plus the outstanding now sum to
		// the live read.
		require.Equal(t, float64(10), resolved[0].Quantity.InexactFloat64())
		require.Equal(t, float64(-6), resolved[1].Quantity.InexactFloat64())
		require.Nil(t, resolved[2].Run)
		require.Equal(t, float64(2), resolved[2].Quantity.InexactFloat64(),
			"outstanding is clamp(realtime - booked cumulative); signed deltas make the realization list reconcile with the live read")
	})
}
