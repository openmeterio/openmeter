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

	discountedUsage, err := ApplyUsageDiscount(ApplyUsageDiscountInput{
		Usage:                 usage,
		RateCardDiscounts:     l.GetRateCardDiscounts(),
		StandardLineDiscounts: l.StandardLineDiscounts,
	})
	if err != nil {
		return l, err
	}

	l.Usage = &discountedUsage.Usage
	l.StandardLineDiscounts = discountedUsage.StandardLineDiscounts

	return l, nil
}

// ApplyUsageDiscountInput describes the raw usage and discounts before usage discount application.
type ApplyUsageDiscountInput struct {
	Usage                 rating.Usage
	RateCardDiscounts     billing.Discounts
	StandardLineDiscounts billing.StandardLineDiscounts
}

// ApplyUsageDiscountResult contains net usage and line-level discount metadata after usage discount application.
type ApplyUsageDiscountResult struct {
	Usage                 rating.Usage
	StandardLineDiscounts billing.StandardLineDiscounts
}

// ApplyUsageDiscount applies the rate card usage discount contract shared by standard billing and charge line projection.
func ApplyUsageDiscount(in ApplyUsageDiscountInput) (ApplyUsageDiscountResult, error) {
	out := ApplyUsageDiscountResult{
		Usage:                 in.Usage,
		StandardLineDiscounts: in.StandardLineDiscounts.Clone(),
	}

	usageDiscount := in.RateCardDiscounts.Usage
	if usageDiscount == nil {
		out.StandardLineDiscounts = removeRateCardUsageDiscounts(out.StandardLineDiscounts)
		return out, nil
	}

	if usageDiscount.CorrelationID == "" {
		return ApplyUsageDiscountResult{}, fmt.Errorf("discount has no correlation ID")
	}

	previouslyAppliedDiscount := applyDiscount(usageDiscount.UsageDiscount.Quantity, in.Usage.PreLinePeriodQuantity)
	out.Usage.PreLinePeriodQuantity = in.Usage.PreLinePeriodQuantity.Sub(previouslyAppliedDiscount.discountApplied)

	discountAppliedToCurrentLine := applyDiscount(previouslyAppliedDiscount.discountRemaining, in.Usage.Quantity)
	if discountAppliedToCurrentLine.discountApplied.IsZero() {
		out.StandardLineDiscounts = removeRateCardUsageDiscounts(out.StandardLineDiscounts)
		return out, nil
	}

	out.StandardLineDiscounts = removeRateCardUsageDiscounts(out.StandardLineDiscounts)
	out.Usage.Quantity = in.Usage.Quantity.Sub(discountAppliedToCurrentLine.discountApplied)
	out.StandardLineDiscounts.Usage = out.StandardLineDiscounts.Usage.MergeDiscountsByChildUniqueReferenceID(
		billing.UsageLineDiscountManaged{
			UsageLineDiscount: billing.UsageLineDiscount{
				LineDiscountBase: billing.LineDiscountBase{
					ChildUniqueReferenceID: lo.ToPtr(fmt.Sprintf(rating.RateCardDiscountChildUniqueReferenceID, usageDiscount.CorrelationID)),
					Reason:                 billing.NewDiscountReasonFrom(*usageDiscount),
				},
				Quantity:              discountAppliedToCurrentLine.discountApplied,
				PreLinePeriodQuantity: lo.EmptyableToPtr(previouslyAppliedDiscount.discountApplied),
			},
		},
	)

	return out, nil
}

func removeRateCardUsageDiscounts(discounts billing.StandardLineDiscounts) billing.StandardLineDiscounts {
	discounts.Usage = lo.Filter(discounts.Usage, func(item billing.UsageLineDiscountManaged, _ int) bool {
		return item.Reason.Type() != billing.RatecardUsageDiscountReason
	})

	return discounts
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
