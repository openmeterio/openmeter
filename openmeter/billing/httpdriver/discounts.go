//go:generate go tool github.com/jmattheis/goverter/cmd/goverter gen ./

package httpdriver

import (
	apiv1 "github.com/openmeterio/openmeter/api"
	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
)

func AsPercentageDiscount(d apiv1models.BillingDiscountPercentage) productcatalog.PercentageDiscount {
	return productcataloghttp.AsPercentageDiscount(apiv1.FromBillingDiscountPercentageToDiscountPercentage(d))
}

func AsUsageDiscount(d apiv1models.BillingDiscountUsage) (productcatalog.UsageDiscount, error) {
	pcUsageDiscount := apiv1.FromBillingDiscountUsageToDiscountUsage(d)

	usageDiscount, err := productcataloghttp.AsUsageDiscount(pcUsageDiscount)
	if err != nil {
		return productcatalog.UsageDiscount{}, err
	}

	return usageDiscount, nil
}

// AsDiscounts converts a BillingDiscounts struct to a productcatalog.Discounts struct.
//
// By converting to productcatalog.Discounts and not billing.Discounts, we drop the DiscountCorrelationID
// field. This prevents random IDs being injected from the clients.
//
// This is not a problem, as for:
//   - Legacy billing this correlation ID is used to link discounts across progressively billed lines,
//     but editing a progressively billed line is not allowed.
//   - Charges only have this field for legacy compatibility, it's value doesn't matter.
func AsDiscounts(discounts *apiv1models.BillingDiscounts) (productcatalog.Discounts, error) {
	out := productcatalog.Discounts{}
	if discounts == nil {
		return out, nil
	}

	if discounts.Percentage != nil {
		pctDiscount := apiv1.FromBillingDiscountPercentageToDiscountPercentage(*discounts.Percentage)

		percentageDiscount := productcataloghttp.AsPercentageDiscount(pctDiscount)
		out.Percentage = &percentageDiscount
	}

	if discounts.Usage != nil {
		uDiscount := apiv1.FromBillingDiscountUsageToDiscountUsage(*discounts.Usage)

		usageDiscount, err := productcataloghttp.AsUsageDiscount(uDiscount)
		if err != nil {
			return productcatalog.Discounts{}, err
		}

		out.Usage = &usageDiscount
	}

	return out, nil
}
