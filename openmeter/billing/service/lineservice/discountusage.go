package lineservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

var _ PricerMiddleware = (*discountUsagePricer)(nil)

// discountUsagePricer applies the unit discounts to the usage based lines.
type discountUsagePricer struct {
	PricerMiddlewareBase
}

func (p *discountUsagePricer) BeforeCalculate(ctx context.Context, l usageBasedLine) (usageBasedLine, error) {
	if l.line.UsageBased.MeteredQuantity == nil {
		return l, fmt.Errorf("usage based line has no quantity")
	}

	if l.line.ParentLineID != nil {
		// TODO: implement
		return l, models.NewGenericNotImplementedError(errors.New("discount usage for progressively billed lines are not implemented"))
	}

	lineQuantityAfterDiscounts := *l.line.UsageBased.MeteredQuantity

	for idx, rcDiscount := range l.line.RateCardDiscounts {
		if rcDiscount.Type() != productcatalog.UsageDiscountType {
			continue
		}

		usageDiscount, err := rcDiscount.AsUsage()
		if err != nil {
			return l, err
		}

		discountQuantity := usageDiscount.Quantity

		if discountQuantity.GreaterThanOrEqual(lineQuantityAfterDiscounts) {
			discountQuantity = lineQuantityAfterDiscounts
			lineQuantityAfterDiscounts = alpacadecimal.Zero
		} else {
			lineQuantityAfterDiscounts = lineQuantityAfterDiscounts.Sub(discountQuantity)
		}

		l.line.Discounts = append(l.line.Discounts, billing.NewLineDiscountFrom(billing.UsageLineDiscount{
			LineDiscountBase: billing.LineDiscountBase{
				ChildUniqueReferenceID: lo.ToPtr(fmt.Sprintf("rateCard/discounts/%d", idx)),
				Reason:                 billing.LineDiscountReasonRatecardDiscount,
				SourceDiscount:         &rcDiscount,
			},
			Quantity: discountQuantity,
		}))
	}

	l.line.UsageBased.Quantity = lo.ToPtr(lineQuantityAfterDiscounts)

	return l, nil
}
