package service

import (
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// resolveFlatFeeRealizations derives the presentation view of a flat-fee
// charge's realization history. Flat fees are never partially invoiced:
// any live run recognizes the whole service period, so coverage is decided
// by run existence, not by period arithmetic. When a live run exists the
// booked history is returned as-is (prior and current runs ordered by
// creation time, voided runs included as inert audit entries); when only
// voided history (or none) exists — unless the charge is final or deleted,
// whose remainder no further run will ever realize — the resolution is a
// single outstanding projection spanning the whole service period, in place
// of the (necessarily voided) history.
func resolveFlatFeeRealizations(charge flatfee.Charge, invoiceLinesByID map[string]billing.StandardInvoice) ([]charges.CustomerChargeFlatFeeRealization, error) {
	status, err := charge.Status.ToMetaChargeStatus()
	if err != nil {
		return nil, fmt.Errorf("converting charge status: %w", err)
	}

	servicePeriod := charge.Intent.GetEffectiveIntent().ServicePeriod

	// A fresh allocation avoids writing into the caller's PriorRuns backing
	// array when the current run is appended.
	runs := make(flatfee.RealizationRuns, 0, len(charge.Realizations.PriorRuns)+1)
	runs = append(runs, charge.Realizations.PriorRuns...)
	if charge.Realizations.CurrentRun != nil {
		runs = append(runs, *charge.Realizations.CurrentRun)
	}

	slices.SortStableFunc(runs, func(a flatfee.RealizationRun, b flatfee.RealizationRun) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	out := make([]charges.CustomerChargeFlatFeeRealization, 0, len(runs)+1)
	hasLiveRun := false

	for idx := range runs {
		run := runs[idx]
		voided := run.IsVoidedBillingHistory()

		entry := charges.CustomerChargeFlatFeeRealization{
			Run:           &runs[idx],
			ServicePeriod: meta.NormalizeClosedPeriod(run.ServicePeriod),
			Voided:        voided,
		}

		if run.LineID != nil {
			if invoice, ok := invoiceLinesByID[*run.LineID]; ok {
				entry.Invoice = &invoice
			}
		}

		out = append(out, entry)

		if !voided {
			hasLiveRun = true
		}
	}

	if status != meta.ChargeStatusFinal && status != meta.ChargeStatusDeleted && !hasLiveRun {
		return []charges.CustomerChargeFlatFeeRealization{{
			ServicePeriod: meta.NormalizeClosedPeriod(servicePeriod),
		}}, nil
	}

	return out, nil
}

// resolveUsageBasedRealizations derives the presentation view of a
// usage-based charge's realization history over its effective service
// period: runs are ordered by creation time (runs snapshot a cumulative
// quantity, so booking order is the meaningful sequence), each live run's
// period starts where the previously created non-voided run ended (the first
// at servicePeriod.From), and its quantity is the cumulative metered
// quantity minus the previous non-voided run's. Voided runs are included as inert
// audit entries. The uncovered tail is projected as an outstanding entry
// spanning from the furthest non-voided service-period end to the effective
// service period end — voided history never counts as coverage, and a
// later-created run with an earlier period end cannot pull the outstanding
// start backwards — unless the charge is final or deleted: those charges
// keep their original service period even when their runs stop short of it,
// and no further run will ever realize the remainder. When the realtime usage expand supplied a
// live metered quantity, the outstanding projection reports the
// not-yet-booked remainder of that read instead of zero.
//
// Booked entries report their delta signed: cumulative snapshots may
// legitimately shrink (downward usage corrections are a supported rating
// scenario, and voided runs may have snapshotted before their live
// neighbors), and a negative quantity states that correction frankly instead
// of fabricating a zero. Only the outstanding projection is floored at zero,
// as the wire contract promises it is never negative. Billing-facing delta
// calculations keep their own strict guards
// (RealizationRuns.MapToBillingMeteredQuantity).
func resolveUsageBasedRealizations(charge usagebased.Charge, invoiceLinesByID map[string]billing.StandardInvoice) ([]charges.CustomerChargeUsageBasedRealization, error) {
	status, err := charge.Status.ToMetaChargeStatus()
	if err != nil {
		return nil, fmt.Errorf("converting charge status: %w", err)
	}

	servicePeriod := charge.Intent.GetEffectiveIntent().ServicePeriod

	// Sorting the charge's own slice in place would mutate domain state the
	// caller still holds.
	runs := slices.Clone(charge.Realizations)

	slices.SortStableFunc(runs, func(a usagebased.RealizationRun, b usagebased.RealizationRun) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	out := make([]charges.CustomerChargeUsageBasedRealization, 0, len(runs)+1)

	coveredUntil := meta.NormalizeTimestamp(servicePeriod.From)
	// maxCoveredTo tracks the furthest service-period end any non-voided run
	// has booked, independent of creation order. The outstanding projection
	// starts there: coveredUntil stitches per-entry periods in creation order
	// and would let a later-created run with an earlier period end pull the
	// outstanding start backwards over already-recognized time.
	maxCoveredTo := meta.NormalizeTimestamp(servicePeriod.From)
	previousQuantity := alpacadecimal.Zero

	for idx := range runs {
		run := runs[idx]
		voided := run.IsVoidedBillingHistory()

		quantity := run.MeteredQuantity.Sub(previousQuantity)

		entry := charges.CustomerChargeUsageBasedRealization{
			Run:           &runs[idx],
			ServicePeriod: timeutil.ClosedPeriod{From: coveredUntil, To: meta.NormalizeTimestamp(run.ServicePeriodTo)},
			Quantity:      quantity,
			Voided:        voided,
		}

		if run.LineID != nil {
			if invoice, ok := invoiceLinesByID[*run.LineID]; ok {
				entry.Invoice = &invoice
			}
		}

		out = append(out, entry)

		if !voided {
			coveredUntil = meta.NormalizeTimestamp(run.ServicePeriodTo)
			if coveredUntil.After(maxCoveredTo) {
				maxCoveredTo = coveredUntil
			}
			previousQuantity = run.MeteredQuantity
		}
	}

	servicePeriodTo := meta.NormalizeTimestamp(servicePeriod.To)
	if status != meta.ChargeStatusFinal && status != meta.ChargeStatusDeleted && maxCoveredTo.Before(servicePeriodTo) {
		outstanding := charges.CustomerChargeUsageBasedRealization{
			ServicePeriod: timeutil.ClosedPeriod{From: maxCoveredTo, To: servicePeriodTo},
			Quantity:      alpacadecimal.Zero,
		}

		// Signed booked deltas telescope to the last non-voided cumulative,
		// so the not-yet-booked remainder of the live read is simply the read
		// minus that cumulative, floored at zero when booked history is ahead
		// of the live read (late-arriving usage the next run will absorb).
		if charge.Expands.RealtimeQuantity != nil {
			realtimeOutstanding := charge.Expands.RealtimeQuantity.Sub(previousQuantity)
			if realtimeOutstanding.IsNegative() {
				realtimeOutstanding = alpacadecimal.Zero
			}

			outstanding.Quantity = realtimeOutstanding
		}

		out = append(out, outstanding)
	}

	return out, nil
}
