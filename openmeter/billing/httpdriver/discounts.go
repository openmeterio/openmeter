//go:generate go tool github.com/jmattheis/goverter/cmd/goverter gen ./

package httpdriver

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
)

func AsPercentageDiscount(d api.BillingDiscountPercentage) billing.PercentageDiscount {
	return billing.PercentageDiscount{
		PercentageDiscount: productcataloghttp.AsPercentageDiscount(api.FromBillingDiscountPercentageToDiscountPercentage(d)),
		CorrelationID:      lo.FromPtr(d.CorrelationId),
	}
}

func AsUsageDiscount(d api.BillingDiscountUsage) (billing.UsageDiscount, error) {
	pcUsageDiscount := api.FromBillingDiscountUsageToDiscountUsage(d)

	usageDiscount, err := productcataloghttp.AsUsageDiscount(pcUsageDiscount)
	if err != nil {
		return billing.UsageDiscount{}, err
	}

	return billing.UsageDiscount{
		UsageDiscount: usageDiscount,
		CorrelationID: lo.FromPtr(d.CorrelationId),
	}, nil
}

func AsDiscounts(discounts *api.BillingDiscounts) (billing.Discounts, error) {
	out := billing.Discounts{}
	if discounts == nil {
		return out, nil
	}

	if discounts.Percentage != nil {
		pctDiscount := api.FromBillingDiscountPercentageToDiscountPercentage(*discounts.Percentage)

		out.Percentage = &billing.PercentageDiscount{
			PercentageDiscount: productcataloghttp.AsPercentageDiscount(pctDiscount),
			CorrelationID:      lo.FromPtr(discounts.Percentage.CorrelationId),
		}
	}

	if discounts.Usage != nil {
		uDiscount := api.FromBillingDiscountUsageToDiscountUsage(*discounts.Usage)

		usageDiscount, err := productcataloghttp.AsUsageDiscount(uDiscount)
		if err != nil {
			return billing.Discounts{}, err
		}

		out.Usage = &billing.UsageDiscount{
			UsageDiscount: usageDiscount,
			CorrelationID: lo.FromPtr(discounts.Usage.CorrelationId),
		}
	}

	return out, nil
}
