package lineservice

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type packagePricer struct {
	ProgressiveBillingPricer
}

var _ Pricer = (*packagePricer)(nil)

func (p packagePricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	packagePrice, err := l.line.UsageBased.Price.AsPackage()
	if err != nil {
		return nil, fmt.Errorf("converting price to package price: %w", err)
	}

	totalUsage := l.linePeriodQty.Add(l.preLinePeriodQty)

	preLinePeriodPackages := p.getNumberOfPackages(l.preLinePeriodQty, packagePrice.QuantityPerPackage)
	if l.IsFirstInPeriod() {
		preLinePeriodPackages = alpacadecimal.Zero
	}

	postLinePeriodPackages := p.getNumberOfPackages(totalUsage, packagePrice.QuantityPerPackage)

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
