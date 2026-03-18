package price

import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var DecimalOne = alpacadecimal.NewFromInt(1)

type ProgressiveBillingMeteredPricer struct{}

func (ProgressiveBillingMeteredPricer) ResolveBillablePeriod(in pricer.ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error) {
	asOf := in.AsOf.Truncate(streaming.MinimumWindowSizeDuration)
	period := in.Line.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration)

	// If progressive billing is not enabled we only bill the line if asof >= line.period.end
	if !in.ProgressiveBilling {
		if asOf.Before(period.To) {
			return nil, nil
		}

		return &period, nil
	}

	// If progressive billing is enabled we only need to make sure that asOf > period.from
	// given we have already truncated the period to the minimum window size duration, this
	// check also makes sure that we have at least 1s of difference, thus usage data in that
	// period.
	if !asOf.After(period.From) {
		return nil, nil
	}

	// If the asOf is before the period end, we need to truncate the period to the asOf
	periodEnd := period.To
	if asOf.Before(periodEnd) {
		periodEnd = asOf
	}

	return &timeutil.ClosedPeriod{
		From: period.From,
		To:   periodEnd,
	}, nil
}

type NonProgressiveBillingPricer struct{}

func (NonProgressiveBillingPricer) ResolveBillablePeriod(in pricer.ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error) {
	// Invoicing a line that has a parent line is not supported, as that's a progressive billing use-case
	//
	// This check is crucial, as when changing a price on a line from progressive billable to non-progressive
	// billable, the CanBeInvoicedAsOf is called to ensure that the line is still valid.
	if in.Line.GetSplitLineGroupID() != nil {
		return nil, billing.ValidationError{
			Err: billing.ErrInvoiceProgressiveBillingNotSupported,
		}
	}

	if in.AsOf.Truncate(streaming.MinimumWindowSizeDuration).Before(in.Line.GetServicePeriod().To.Truncate(streaming.MinimumWindowSizeDuration)) {
		return nil, nil
	}

	return lo.ToPtr(in.Line.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration)), nil
}
