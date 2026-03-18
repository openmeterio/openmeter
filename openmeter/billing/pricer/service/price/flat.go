package price

import (
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Flat struct{}

var _ Pricer = (*Flat)(nil)

func (p Flat) GenerateDetailedLines(l PricerCalculateInput) (pricer.DetailedLines, error) {
	flatPrice, err := l.GetPrice().AsFlat()
	if err != nil {
		return nil, fmt.Errorf("converting price to flat price: %w", err)
	}

	if flatPrice.PaymentTerm == "" {
		flatPrice.PaymentTerm = productcatalog.DefaultPaymentTerm
	}

	if !slices.Contains(
		[]productcatalog.PaymentTermType{productcatalog.InAdvancePaymentTerm, productcatalog.InArrearsPaymentTerm},
		flatPrice.PaymentTerm) {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("flat price payment term %s is not supported", flatPrice.PaymentTerm),
		}
	}

	switch {
	case flatPrice.PaymentTerm == productcatalog.InAdvancePaymentTerm && l.IsFirstInPeriod():
		return pricer.DetailedLines{
			{
				Name:                   l.GetName(),
				Quantity:               alpacadecimal.NewFromInt(1),
				PerUnitAmount:          flatPrice.Amount,
				ChildUniqueReferenceID: pricer.FlatPriceChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InAdvancePaymentTerm,
			},
		}, nil
	case flatPrice.PaymentTerm == productcatalog.InArrearsPaymentTerm && l.IsLastInPeriod():
		return pricer.DetailedLines{
			{
				Name:                   l.GetName(),
				Quantity:               alpacadecimal.NewFromInt(1),
				PerUnitAmount:          flatPrice.Amount,
				ChildUniqueReferenceID: pricer.FlatPriceChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}, nil
	}

	return nil, nil
}

func (p Flat) ResolveBillablePeriod(in pricer.ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error) {
	if in.Line.GetSplitLineGroupID() != nil {
		return nil, billing.ValidationError{
			Err: billing.ErrInvoiceProgressiveBillingNotSupported,
		}
	}

	// For the flat prices they are always billable but the invoiceAt signifies when the line should be
	// actually billed.
	invoiceAtTruncated := in.Line.GetInvoiceAt().Truncate(streaming.MinimumWindowSizeDuration)
	asOfTruncated := in.AsOf.Truncate(streaming.MinimumWindowSizeDuration)

	if invoiceAtTruncated.After(asOfTruncated) {
		return nil, nil
	}

	return lo.ToPtr(in.Line.GetServicePeriod()), nil
}
