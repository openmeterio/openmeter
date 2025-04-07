package lineservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type unitPricer struct{}

var _ Pricer = (*unitPricer)(nil)

func (p *unitPricer) Calculate(ctx context.Context, l usageBasedLine) (newDetailedLinesInput, error) {
	if l.line.UsageBased.Quantity == nil || l.line.UsageBased.Quantity.IsZero() {
		return nil, nil
	}

	unitPrice, err := l.line.UsageBased.Price.AsUnit()
	if err != nil {
		return nil, err
	}

	return newDetailedLinesInput{
		{
			Name:                   fmt.Sprintf("%s: usage in period", l.line.Name),
			Quantity:               *l.line.UsageBased.Quantity,
			PerUnitAmount:          unitPrice.Amount,
			ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
			PaymentTerm:            productcatalog.InArrearsPaymentTerm,
		},
	}, nil
}

func (p *unitPricer) Capabilities(l usageBasedLine) (PricerCapabilities, error) {
	// TODO: filter by meter type

	return PricerCapabilities{
		AllowsProgressiveBilling: true,
	}, nil
}
