package lineservice

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type unitPricer struct {
	ProgressiveBillingMeteredPricer
}

var _ Pricer = (*unitPricer)(nil)

func (p unitPricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	unitPrice, err := l.line.UsageBased.Price.AsUnit()
	if err != nil {
		return nil, fmt.Errorf("converting price to unit price: %w", err)
	}

	usage, err := l.GetUsage()
	if err != nil {
		return nil, err
	}

	if usage.LinePeriodQuantity.IsPositive() {
		return newDetailedLinesInput{
			{
				Name:                   fmt.Sprintf("%s: usage in period", l.line.Name),
				Quantity:               usage.LinePeriodQuantity,
				PerUnitAmount:          unitPrice.Amount,
				ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}, nil
	}

	return nil, nil
}
