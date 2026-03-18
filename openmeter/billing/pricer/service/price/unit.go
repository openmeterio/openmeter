package price

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/pricer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type Unit struct {
	ProgressiveBillingMeteredPricer
}

var _ Pricer = (*Unit)(nil)

func (p Unit) GenerateDetailedLines(l PricerCalculateInput) (pricer.DetailedLines, error) {
	unitPrice, err := l.GetPrice().AsUnit()
	if err != nil {
		return nil, fmt.Errorf("converting price to unit price: %w", err)
	}

	usage, err := l.GetUsage()
	if err != nil {
		return nil, err
	}

	if usage.Quantity.IsPositive() {
		return pricer.DetailedLines{
			{
				Name:                   fmt.Sprintf("%s: usage in period", l.GetName()),
				Quantity:               usage.Quantity,
				PerUnitAmount:          unitPrice.Amount,
				ChildUniqueReferenceID: pricer.UnitPriceUsageChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}, nil
	}

	return nil, nil
}
