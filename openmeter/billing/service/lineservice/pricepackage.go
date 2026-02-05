package lineservice

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type packagePricer struct {
	ProgressiveBillingMeteredPricer
}

var _ Pricer = (*packagePricer)(nil)

func (p packagePricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	usage, err := l.GetUsage()
	if err != nil {
		return nil, err
	}

	packagePrice, err := l.line.UsageBased.Price.AsPackage()
	if err != nil {
		return nil, fmt.Errorf("converting price to package price: %w", err)
	}

	totalUsage := usage.LinePeriodQuantity.Add(usage.PreLinePeriodQuantity)

	preLinePeriodPackages := p.getNumberOfPackages(usage.PreLinePeriodQuantity, packagePrice.QuantityPerPackage)
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
