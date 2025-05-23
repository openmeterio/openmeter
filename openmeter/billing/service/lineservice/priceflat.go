package lineservice

import (
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type flatPricer struct {
	// TODO[later]: when we introduce full pro-rating support this should be ProgressiveBillingPricer
	NonProgressiveBillingPricer
}

var _ Pricer = (*flatPricer)(nil)

func (p flatPricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	flatPrice, err := l.line.UsageBased.Price.AsFlat()
	if err != nil {
		return nil, fmt.Errorf("converting price to flat price: %w", err)
	}

	if !slices.Contains(
		[]productcatalog.PaymentTermType{productcatalog.InAdvancePaymentTerm, productcatalog.InArrearsPaymentTerm},
		flatPrice.PaymentTerm) {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("flat price payment term %s is not supported", flatPrice.PaymentTerm),
		}
	}

	if l.line.UsageBased.Quantity == nil {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("usage based flat price quantity is not set"),
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
