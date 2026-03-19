package rate

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type Package struct {
	ProgressiveBillingMeteredPricer
}

var _ Pricer = (*Package)(nil)

func (p Package) GenerateDetailedLines(l PricerCalculateInput) (rating.DetailedLines, error) {
	usage, err := l.GetUsage()
	if err != nil {
		return nil, err
	}

	packagePrice, err := l.GetPrice().AsPackage()
	if err != nil {
		return nil, fmt.Errorf("converting price to package price: %w", err)
	}

	totalUsage := usage.Quantity.Add(usage.PreLinePeriodQuantity)

	preLinePeriodPackages := p.GetNumberOfPackages(usage.PreLinePeriodQuantity, packagePrice.QuantityPerPackage)
	if l.IsFirstInPeriod() {
		preLinePeriodPackages = alpacadecimal.Zero
	}

	postLinePeriodPackages := p.GetNumberOfPackages(totalUsage, packagePrice.QuantityPerPackage)

	toBeBilledPackages := postLinePeriodPackages.Sub(preLinePeriodPackages)

	if !toBeBilledPackages.IsZero() {
		return rating.DetailedLines{
			{
				Name:                   fmt.Sprintf("%s: usage in period", l.GetName()),
				Quantity:               toBeBilledPackages,
				PerUnitAmount:          packagePrice.Amount,
				ChildUniqueReferenceID: rating.UsageChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}, nil
	}

	return nil, nil
}

func (p Package) GetNumberOfPackages(qty alpacadecimal.Decimal, packageSize alpacadecimal.Decimal) alpacadecimal.Decimal {
	requiredPackages := qty.Div(packageSize).Floor()

	if qty.Mod(packageSize).IsZero() {
		return requiredPackages
	}

	return requiredPackages.Add(DecimalOne)
}
