package lineservice

import (
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type flatPricer struct{}

var _ Pricer = (*flatPricer)(nil)

func (p flatPricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	flatPrice, err := l.line.UsageBased.Price.AsFlat()
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
		return newDetailedLinesInput{
			{
				Name:                   l.line.Name,
				Quantity:               alpacadecimal.NewFromInt(1),
				PerUnitAmount:          flatPrice.Amount,
				ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InAdvancePaymentTerm,
			},
		}, nil
	case flatPrice.PaymentTerm == productcatalog.InArrearsPaymentTerm && l.IsLastInPeriod():
		return newDetailedLinesInput{
			{
				Name:                   l.line.Name,
				Quantity:               alpacadecimal.NewFromInt(1),
				PerUnitAmount:          flatPrice.Amount,
				ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}, nil
	}

	return nil, nil
}

func (p flatPricer) CanBeInvoicedAsOf(in CanBeInvoicedAsOfInput) (*timeutil.ClosedPeriod, error) {
	if in.Line.GetSplitLineGroupID() != nil {
		return nil, billing.ValidationError{
			Err: billing.ErrInvoiceProgressiveBillingNotSupported,
		}
	}

	price := in.Line.GetPrice()
	if price == nil {
		return nil, fmt.Errorf("price is nil")
	}

	if price.Type() != productcatalog.FlatPriceType {
		return nil, fmt.Errorf("price is not a flat price")
	}

	flatPrice, err := price.AsFlat()
	if err != nil {
		return nil, fmt.Errorf("converting price to flat price: %w", err)
	}

	paymentTerm := flatPrice.PaymentTerm
	if paymentTerm == "" {
		paymentTerm = productcatalog.DefaultPaymentTerm
	}

	// canBeInvoicedAt is the time at which the line can be invoiced, this is not
	// necessarily equals to the invoiceAt field of the line, as subscription sync can choose
	// to defer the invoicing to a later time.
	var canBeInvoicedAt time.Time
	switch paymentTerm {
	case productcatalog.InAdvancePaymentTerm:
		canBeInvoicedAt = in.Line.GetServicePeriod().From
	case productcatalog.InArrearsPaymentTerm:
		canBeInvoicedAt = in.Line.GetServicePeriod().To
	default:
		return nil, fmt.Errorf("unsupported payment term: %s", paymentTerm)
	}

	if in.AsOf.Before(canBeInvoicedAt) {
		return nil, nil
	}

	return lo.ToPtr(in.Line.GetServicePeriod()), nil
}
