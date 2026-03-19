package mutator

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
)

type DiscountUsage struct{}

var _ PreCalculationMutator = (*DiscountUsage)(nil)

func (m *DiscountUsage) Mutate(l rate.PricerCalculateInput) (rate.PricerCalculateInput, error) {
	// Warning: This mutator cannot be idempotent, as it manipulates the usage based line directly, not
	// the detailed lines (that are regenerated from the usage based line on each calculation run).
	//
	// This means that we need to ensure that the mutator always keeps the line discounts in sync with the
	// usage based line's rate card discounts.

	usage, err := l.GetUsage()
	if err != nil {
		return l, err
	}

	usageDiscount, err := m.getUsageDiscount(l)
	if err != nil {
		return l, err
	}

	if usageDiscount == nil {
		// If there is no usage discount intent, let's remove all the usage discounts from the line (in case there are any)

		return m.removeUsageDiscounts(l), nil
	}

	discountLimit := usageDiscount.UsageDiscount.Quantity

	previouslyAppliedDiscount := applyDiscount(discountLimit, usage.PreLinePeriodQuantity)
	// Let's apply any previous applied discounts to the pre line period quantity
	l.Usage.PreLinePeriodQuantity = usage.PreLinePeriodQuantity.Sub(previouslyAppliedDiscount.discountApplied)

	// Let's apply the discount to the current line
	discountAppliedToCurrentLine := applyDiscount(previouslyAppliedDiscount.discountRemaining, usage.Quantity)

	if discountAppliedToCurrentLine.discountApplied.IsZero() {
		// We have no discount to apply, let's remove the usage discount from the line

		return m.removeUsageDiscounts(l), nil
	}

	// We have discounts to apply
	l = m.removeUsageDiscounts(l)

	l.Usage.Quantity = usage.Quantity.Sub(discountAppliedToCurrentLine.discountApplied)

	l.StandardLineDiscounts.Usage = l.StandardLineDiscounts.Usage.MergeDiscountsByChildUniqueReferenceID(
		billing.UsageLineDiscountManaged{
			UsageLineDiscount: billing.UsageLineDiscount{
				LineDiscountBase: billing.LineDiscountBase{
					ChildUniqueReferenceID: lo.ToPtr(usageDiscount.childUniqueReferenceID),
					Reason:                 billing.NewDiscountReasonFrom(usageDiscount.UsageDiscount),
				},
				Quantity:              discountAppliedToCurrentLine.discountApplied,
				PreLinePeriodQuantity: lo.EmptyableToPtr(previouslyAppliedDiscount.discountApplied),
			},
		},
	)

	return l, nil
}

func (m *DiscountUsage) removeUsageDiscounts(l rate.PricerCalculateInput) rate.PricerCalculateInput {
	l.StandardLineDiscounts.Usage = lo.Filter(l.StandardLineDiscounts.Usage, func(item billing.UsageLineDiscountManaged, _ int) bool {
		return item.Reason.Type() != billing.RatecardUsageDiscountReason
	})

	return l
}

type usageDiscountWithChildUniqueReferenceID struct {
	billing.UsageDiscount
	childUniqueReferenceID string
}

func (m *DiscountUsage) getUsageDiscount(l rate.PricerCalculateInput) (*usageDiscountWithChildUniqueReferenceID, error) {
	discounts := l.GetRateCardDiscounts()
	if discounts.Usage == nil {
		return nil, nil
	}

	rcUsageDiscount := discounts.Usage

	if rcUsageDiscount.CorrelationID == "" {
		return nil, fmt.Errorf("discount has no correlation ID")
	}

	return &usageDiscountWithChildUniqueReferenceID{
		UsageDiscount:          *rcUsageDiscount,
		childUniqueReferenceID: fmt.Sprintf(rating.RateCardDiscountChildUniqueReferenceID, rcUsageDiscount.CorrelationID),
	}, nil
}

type applyDiscountResult struct {
	discountRemaining alpacadecimal.Decimal
	discountApplied   alpacadecimal.Decimal
}

func applyDiscount(discountAvailable alpacadecimal.Decimal, usage alpacadecimal.Decimal) applyDiscountResult {
	if discountAvailable.LessThan(usage) {
		return applyDiscountResult{
			discountRemaining: alpacadecimal.Zero,
			discountApplied:   discountAvailable,
		}
	}

	return applyDiscountResult{
		discountRemaining: discountAvailable.Sub(usage),
		discountApplied:   usage,
	}
}
