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

// resolveFlatFeeRealizations presents a flat-fee charge's realization
// history as the API sees it: every run in booking order, voided runs kept
// as inert audit entries, with the booked invoice attached when expanded.
//
// A flat fee is never partially invoiced: one live run realizes the whole
// service period. A charge with no live run is therefore wholly outstanding
// and is presented as a single outstanding entry instead of its (necessarily
// voided) history — except final and deleted charges, whose remainder nothing
// will ever realize.
func resolveFlatFeeRealizations(charge flatfee.Charge, invoiceLinesByID map[string]billing.StandardInvoice) ([]charges.CustomerChargeFlatFeeRealization, error) {
	status, err := charge.Status.ToMetaChargeStatus()
	if err != nil {
		return nil, fmt.Errorf("converting charge status: %w", err)
	}

	servicePeriod := charge.Intent.GetEffectiveIntent().ServicePeriod

	// Copied so appending the current run does not write into the caller's
	// PriorRuns backing array.
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

// resolveUsageBasedRealizations presents a usage-based charge's realization
// history as the API sees it: every run in booking order, voided runs kept
// as inert audit entries, and an outstanding entry for whatever part of the
// service period is still unrealized.
//
// Runs are cumulative snapshots, so each entry reports only the usage it
// realized since the previous live run; that delta stays signed because a
// downward usage correction is a legitimate billing event.
// Voided runs never count as realized coverage. Final and deleted charges
// have no outstanding entry: nothing will ever realize their remainder.
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
	// maxCoveredTo is the furthest period end any live run has booked,
	// independent of booking order. The outstanding entry starts there so a
	// later run with an earlier period end cannot move it back over
	// already-realized time.
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

		// The unrealized remainder is the live read minus the last live
		// cumulative, floored at zero when booked history is ahead of the
		// read.
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
