package lineservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type dynamicPricer struct{}

var _ Pricer = (*dynamicPricer)(nil)

func (p *dynamicPricer) Calculate(ctx context.Context, l usageBasedLine) (newDetailedLinesInput, error) {
	if l.line.UsageBased.Quantity == nil {
		return nil, errors.New("usage based line has no quantity")
	}

	if l.line.UsageBased.Quantity.IsPositive() {
		dynamicPrice, err := l.line.UsageBased.Price.AsDynamic()
		if err != nil {
			return nil, err
		}

		amountInPeriod := l.currency.RoundToPrecision(
			l.line.UsageBased.Quantity.Mul(dynamicPrice.Multiplier),
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

func (p *dynamicPricer) Capabilities(l usageBasedLine) (PricerCapabilities, error) {
	return PricerCapabilities{
		AllowsProgressiveBilling: true,
	}, nil
}
