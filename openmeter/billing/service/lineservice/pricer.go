package lineservice

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type PricerCapabilities struct {
	// SupportsProgressiveBilling is true if the given pricer can support progressive billing
	SupportsProgressiveBilling bool
}

type PricerCalculateInput usageBasedLine

type usage struct {
	LinePeriodQuantity    alpacadecimal.Decimal
	PreLinePeriodQuantity alpacadecimal.Decimal
}

func (i PricerCalculateInput) GetUsage() (usage, error) {
	empty := usage{}

	if i.line.Quantity == nil {
		return empty, fmt.Errorf("usage based line[%s]: quantity is nil", i.line.ID)
	}

	if i.line.PreLinePeriodQuantity == nil {
		return empty, fmt.Errorf("usage based line[%s]: pre line period quantity is nil", i.line.ID)
	}

	return usage{
		LinePeriodQuantity:    *i.line.Quantity,
		PreLinePeriodQuantity: *i.line.PreLinePeriodQuantity,
	}, nil
}

func (i PricerCalculateInput) LinePeriodQuantity() alpacadecimal.Decimal {
	return *i.line.Quantity
}

type PricerCanBeInvoicedAsOfAccessor interface {
	PriceAccessor
	GetSplitLineGroupID() *string
	GetInvoiceAt() time.Time
	GetID() string
}

type CanBeInvoicedAsOfInput struct {
	AsOf               time.Time
	ProgressiveBilling bool
	Line               PricerCanBeInvoicedAsOfAccessor
	FeatureMeters      billing.FeatureMeters
}

func (i CanBeInvoicedAsOfInput) Validate() error {
	var errs []error

	if i.Line == nil {
		errs = append(errs, fmt.Errorf("line is required"))
	}

	if i.FeatureMeters == nil {
		errs = append(errs, fmt.Errorf("feature meters are required"))
	}

	if i.AsOf.IsZero() {
		errs = append(errs, fmt.Errorf("as of is required"))
	}

	return errors.Join(errs...)
}

type Pricer interface {
	// Calculate calculates the detailed lines for a line.
	Calculate(line PricerCalculateInput) (newDetailedLinesInput, error)

	// CanBeInvoicedAsOf checks if the line can be invoiced as of the given time and returns the service
	// period that can be invoiced.
	CanBeInvoicedAsOf(CanBeInvoicedAsOfInput) (*timeutil.ClosedPeriod, error)
}

type ProgressiveBillingMeteredPricer struct{}

func (ProgressiveBillingMeteredPricer) CanBeInvoicedAsOf(in CanBeInvoicedAsOfInput) (*timeutil.ClosedPeriod, error) {
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

func (NonProgressiveBillingPricer) CanBeInvoicedAsOf(in CanBeInvoicedAsOfInput) (*timeutil.ClosedPeriod, error) {
	// Invoicing a line that has a parent line is not supported, as that's a progressive billing use-case
	//
	// This check is crucial, as when changing a price on a line from progressive billable to non-progressive
	// billable, the CanBeInvoicedAsOf is called to ensure that the line is still valid.
	if in.Line.GetSplitLineGroupID() != nil {
		return nil, billing.ValidationError{
			Err: billing.ErrInvoiceProgressiveBillingNotSupported,
		}
	}

	if in.AsOf.Before(in.Line.GetServicePeriod().To) {
		return nil, nil
	}

	return lo.ToPtr(in.Line.GetServicePeriod()), nil
}
