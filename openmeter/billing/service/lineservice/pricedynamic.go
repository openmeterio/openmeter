package lineservice

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type dynamicPricer struct {
	ProgressiveBillingPricer
}

var _ Pricer = (*dynamicPricer)(nil)

func (p dynamicPricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	dynamicPrice, err := l.line.UsageBased.Price.AsDynamic()
	if err != nil {
		return nil, fmt.Errorf("converting price to dynamic price: %w", err)
	}

	if l.linePeriodQty.IsPositive() {
		amountInPeriod := l.currency.RoundToPrecision(
			l.linePeriodQty.Mul(dynamicPrice.Multiplier),
		)

		return newDetailedLinesInput{
			{
				Name:                   fmt.Sprintf("%s: usage in period", l.line.Name),
				Quantity:               alpacadecimal.NewFromInt(1),
				PerUnitAmount:          amountInPeriod,
				ChildUniqueReferenceID: UsageChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}, nil
	}

	return nil, nil
}
