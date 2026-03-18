package price

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/pricer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type Dynamic struct {
	ProgressiveBillingMeteredPricer
}

var _ Pricer = (*Dynamic)(nil)

func (p Dynamic) GenerateDetailedLines(l PricerCalculateInput) (pricer.DetailedLines, error) {
	usage, err := l.GetUsage()
	if err != nil {
		return nil, err
	}

	dynamicPrice, err := l.GetPrice().AsDynamic()
	if err != nil {
		return nil, fmt.Errorf("converting price to dynamic price: %w", err)
	}

	if usage.Quantity.IsPositive() {
		amountInPeriod := l.CurrencyCalculator.RoundToPrecision(
			usage.Quantity.Mul(dynamicPrice.Multiplier),
		)

		return pricer.DetailedLines{
			{
				Name:                   fmt.Sprintf("%s: usage in period", l.GetName()),
				Quantity:               alpacadecimal.NewFromInt(1),
				PerUnitAmount:          amountInPeriod,
				ChildUniqueReferenceID: pricer.UsageChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}, nil
	}

	return nil, nil
}
