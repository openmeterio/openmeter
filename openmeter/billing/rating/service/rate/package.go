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

// GetNumberOfPackages returns the number of packages needed to cover qty, rounding up.
//
// The quotient and the remainder come from one QuoRem, so both are the same exact
// division. Div rounds the quotient to DivisionPrecision digits, so Div().Floor() can
// sit on a whole number while the exact Mod still reports a remainder, and the function
// then adds a package that is not owed.
//
// packageSize is validated non-zero and non-negative, and qty is clamped non-negative,
// so the truncating quotient is the floor.
func (p Package) GetNumberOfPackages(qty alpacadecimal.Decimal, packageSize alpacadecimal.Decimal) alpacadecimal.Decimal {
	requiredPackages, remainder := qty.QuoRem(packageSize, 0)

	if remainder.IsZero() {
		return requiredPackages
	}

	return requiredPackages.Add(DecimalOne)
}
