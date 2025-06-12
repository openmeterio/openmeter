package lineservice

import (
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
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

	if i.line.UsageBased.Quantity == nil {
		return empty, fmt.Errorf("usage based line[%s]: quantity is nil", i.line.ID)
	}

	if i.line.UsageBased.PreLinePeriodQuantity == nil {
		return empty, fmt.Errorf("usage based line[%s]: pre line period quantity is nil", i.line.ID)
	}

	return usage{
		LinePeriodQuantity:    *i.line.UsageBased.Quantity,
		PreLinePeriodQuantity: *i.line.UsageBased.PreLinePeriodQuantity,
	}, nil
}

func (i PricerCalculateInput) LinePeriodQuantity() alpacadecimal.Decimal {
	return *i.line.UsageBased.Quantity
}

type Pricer interface {
	// Calculate calculates the detailed lines for a line.
	Calculate(line PricerCalculateInput) (newDetailedLinesInput, error)

	// CanBeInvoicedAsOf checks if the line can be invoiced as of the given time.
	CanBeInvoicedAsOf(usageBasedLine, time.Time) (bool, error)
}

type ProgressiveBillingPricer struct{}

func (ProgressiveBillingPricer) CanBeInvoicedAsOf(l usageBasedLine, asOf time.Time) (bool, error) {
	if asOf.Before(l.line.Period.Start) {
		return false, nil
	}

	return true, nil
}

type NonProgressiveBillingPricer struct{}

func (NonProgressiveBillingPricer) CanBeInvoicedAsOf(l usageBasedLine, asOf time.Time) (bool, error) {
	// Invoicing a line that has a parent line is not supported, as that's a progressive billing use-case
	//
	// This check is crucial, as when changing a price on a line from progressive billable to non-progressive
	// billable, the CanBeInvoicedAsOf is called to ensure that the line is still valid.
	if l.line.SplitLineGroupID != nil {
		return false, billing.ValidationError{
			Err: billing.ErrInvoiceProgressiveBillingNotSupported,
		}
	}

	if asOf.Before(l.line.Period.End) {
		return false, nil
	}

	return true, nil
}
