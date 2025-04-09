//go:generate go run github.com/jmattheis/goverter/cmd/goverter gen ./

package httpdriver

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
)

func AsPercentageDiscount(d api.BillingDiscountPercentage) billing.PercentageDiscount {
	return billing.PercentageDiscount{
		PercentageDiscount: productcataloghttp.AsPercentageDiscount(api.FromBillingDiscountPercentageToDiscountPercentage(d)),
		CorrelationID:      lo.FromPtrOr(d.CorrelationId, ""),
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
		CorrelationID: lo.FromPtrOr(d.CorrelationId, ""),
	}, nil
}

func AsDiscounts(discounts []api.BillingDiscount) (billing.Discounts, error) {
	out := make(billing.Discounts, 0, len(discounts))
	for _, d := range discounts {
		disc, err := d.Discriminator()
		if err != nil {
			return nil, err
		}

		switch disc {
		case string(api.BillingDiscountPercentageTypePercentage):
			pctDiscount, err := d.AsBillingDiscountPercentage()
			if err != nil {
				return nil, err
			}

			out = append(out, billing.NewDiscountFrom(AsPercentageDiscount(pctDiscount)))
		case string(api.BillingDiscountUsageTypeUsage):
			usageDiscountAPI, err := d.AsBillingDiscountUsage()
			if err != nil {
				return nil, err
			}

			usageDiscount, err := AsUsageDiscount(usageDiscountAPI)
			if err != nil {
				return nil, err
			}

			out = append(out, billing.NewDiscountFrom(usageDiscount))
		default:
			return nil, fmt.Errorf("invalid discount type: %s", disc)
		}
	}
	return out, nil
}
