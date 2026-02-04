package lineservice

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type dynamicPricer struct {
	ProgressiveBillingMeteredPricer
}

var _ Pricer = (*dynamicPricer)(nil)

func (p dynamicPricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	usage, err := l.GetUsage()
	if err != nil {
		return nil, err
	}

	dynamicPrice, err := l.line.UsageBased.Price.AsDynamic()
	if err != nil {
		return nil, fmt.Errorf("converting price to dynamic price: %w", err)
	}

	if usage.LinePeriodQuantity.IsPositive() {
		amountInPeriod := l.currency.RoundToPrecision(
			usage.LinePeriodQuantity.Mul(dynamicPrice.Multiplier),
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
