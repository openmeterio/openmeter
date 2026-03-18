package rate

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type Unit struct {
	ProgressiveBillingMeteredPricer
}

var _ Pricer = (*Unit)(nil)

func (p Unit) GenerateDetailedLines(l PricerCalculateInput) (rating.DetailedLines, error) {
	unitPrice, err := l.GetPrice().AsUnit()
	if err != nil {
		return nil, fmt.Errorf("converting price to unit price: %w", err)
	}

	usage, err := l.GetUsage()
	if err != nil {
		return nil, err
	}

	if usage.Quantity.IsPositive() {
		return rating.DetailedLines{
			{
				Name:                   fmt.Sprintf("%s: usage in period", l.GetName()),
				Quantity:               usage.Quantity,
				PerUnitAmount:          unitPrice.Amount,
				ChildUniqueReferenceID: rating.UnitPriceUsageChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}, nil
	}

	return nil, nil
}
