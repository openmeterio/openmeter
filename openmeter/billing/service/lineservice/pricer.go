package lineservice

import (
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type PricerCapabilities struct {
	// SupportsProgressiveBilling is true if the given pricer can support progressive billing
	SupportsProgressiveBilling bool
}

type PricerCalculateInput struct {
	usageBasedLine

	preLinePeriodQty alpacadecimal.Decimal
	linePeriodQty    alpacadecimal.Decimal
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
	if l.line.ParentLineID != nil {
		return false, billing.ValidationError{
			Err: billing.ErrInvoiceProgressiveBillingNotSupported,
		}
	}

	if asOf.Before(l.line.Period.End) {
		return false, nil
	}

	return true, nil
}
