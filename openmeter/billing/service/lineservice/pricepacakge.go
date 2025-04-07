package lineservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type packagePricer struct{}

var _ Pricer = (*packagePricer)(nil)

func (p packagePricer) Calculate(ctx context.Context, l usageBasedLine) (newDetailedLinesInput, error) {
	if l.line.UsageBased.Quantity == nil {
		return nil, errors.New("usage based line has no quantity")
	}

	packagePrice, err := l.line.UsageBased.Price.AsPackage()
	if err != nil {
		return nil, err
	}

	currentPeriodQty := l.line.UsageBased.Quantity
	preLinePeriodQty := alpacadecimal.Zero

	if l.line.UsageBased.PreLinePeriodQuantity != nil {
		preLinePeriodQty = *l.line.UsageBased.PreLinePeriodQuantity
	}

	totalUsage := currentPeriodQty.Add(preLinePeriodQty)

	preLinePeriodPackages := l.getNumberOfPackages(preLinePeriodQty, packagePrice.QuantityPerPackage)
	if l.IsFirstInPeriod() {
		preLinePeriodPackages = alpacadecimal.Zero
	}

	postLinePeriodPackages := l.getNumberOfPackages(totalUsage, packagePrice.QuantityPerPackage)

	toBeBilledPackages := postLinePeriodPackages.Sub(preLinePeriodPackages)

	if !toBeBilledPackages.IsZero() {
		return newDetailedLinesInput{
			{
				Name:                   fmt.Sprintf("%s: usage in period", l.line.Name),
				Quantity:               toBeBilledPackages,
				PerUnitAmount:          packagePrice.Amount,
				ChildUniqueReferenceID: UsageChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}, nil
	}

	return nil, nil
}

func (p packagePricer) getNumberOfPackages(qty alpacadecimal.Decimal, packageSize alpacadecimal.Decimal) alpacadecimal.Decimal {
	requiredPackages := qty.Div(packageSize).Floor()

	if qty.Mod(packageSize).IsZero() {
		return requiredPackages
	}

	return requiredPackages.Add(DecimalOne)
}

func (p packagePricer) Capabilities(l usageBasedLine) (PricerCapabilities, error) {
	return PricerCapabilities{
		AllowsProgressiveBilling: true,
	}, nil
}
