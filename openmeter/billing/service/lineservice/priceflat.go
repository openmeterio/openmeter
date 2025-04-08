package lineservice

import (
	"fmt"

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

	switch {
	case flatPrice.PaymentTerm == productcatalog.InAdvancePaymentTerm && l.IsFirstInPeriod():
		return newDetailedLinesInput{
			{
				Name:                   l.line.Name,
				Quantity:               alpacadecimal.NewFromFloat(1),
				PerUnitAmount:          flatPrice.Amount,
				ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InAdvancePaymentTerm,
			},
		}, nil
	case flatPrice.PaymentTerm == productcatalog.InArrearsPaymentTerm && l.IsLastInPeriod():
		return newDetailedLinesInput{
			{
				Name:                   l.line.Name,
				Quantity:               alpacadecimal.NewFromFloat(1),
				PerUnitAmount:          flatPrice.Amount,
				ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}, nil
	default:
		return nil, billing.ValidationError{
			Err: fmt.Errorf("flat price payment term %s is not supported", flatPrice.PaymentTerm),
		}
	}
}
