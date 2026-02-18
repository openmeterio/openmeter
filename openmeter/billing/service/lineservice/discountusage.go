package lineservice

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type discountUsageMutator struct{}

var _ PreCalculationMutator = (*discountUsageMutator)(nil)

func (m *discountUsageMutator) Mutate(l PricerCalculateInput) (PricerCalculateInput, error) {
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

	discountBeforeLineUsedQty := alpacadecimal.Zero
	if l.line.SplitLineGroupID != nil {
		usedQty, err := m.calculateUsedQtyByCorrelationID(l.line, usageDiscount.UsageDiscount.CorrelationID)
		if err != nil {
			return l, err
		}

		discountBeforeLineUsedQty = usedQty
	}

	discountLimit := usageDiscount.UsageDiscount.Quantity

	discountRemaining := discountLimit.Sub(discountBeforeLineUsedQty)
	if discountRemaining.LessThan(alpacadecimal.Zero) {
		// We have already exhausted the discount quantity.

		// This should not happen, but in case of a progressive billing scenario, if the discount is edited on some of the
		// lines, then we can end up in such a state.
		return l, fmt.Errorf("usage discount quantity on line[%s] is overcommitted by %s", l.line.ID, discountRemaining.Neg())
	}

	// Let's calculate the amount of discount to apply
	discountQuantity := usage.LinePeriodQuantity
	if discountRemaining.LessThan(usage.LinePeriodQuantity) {
		discountQuantity = discountRemaining
	}

	if discountQuantity.LessThanOrEqual(alpacadecimal.Zero) {
		// We have no discount to apply, let's remove the usage discount from the line

		// We still need to set the pre line period quantity to the remaining quantity, to be consistent with the
		// usage discount case.
		l.line.PreLinePeriodQuantity = lo.ToPtr(usage.PreLinePeriodQuantity.Sub(discountBeforeLineUsedQty))

		return m.removeUsageDiscounts(l), nil
	}

	l.line.Discounts.Usage = l.line.Discounts.Usage.MergeDiscountsByChildUniqueReferenceID(
		billing.UsageLineDiscountManaged{
			UsageLineDiscount: billing.UsageLineDiscount{
				LineDiscountBase: billing.LineDiscountBase{
					ChildUniqueReferenceID: lo.ToPtr(usageDiscount.childUniqueReferenceID),
					Reason:                 billing.NewDiscountReasonFrom(usageDiscount.UsageDiscount),
				},
				Quantity:              discountQuantity,
				PreLinePeriodQuantity: lo.EmptyableToPtr(discountBeforeLineUsedQty),
			},
		},
	)

	l.line.Quantity = lo.ToPtr(usage.LinePeriodQuantity.Sub(discountQuantity))
	l.line.PreLinePeriodQuantity = lo.ToPtr(usage.PreLinePeriodQuantity.Sub(discountBeforeLineUsedQty))

	return l, nil
}

func (m *discountUsageMutator) removeUsageDiscounts(l PricerCalculateInput) PricerCalculateInput {
	l.line.Discounts.Usage = lo.Filter(l.line.Discounts.Usage, func(item billing.UsageLineDiscountManaged, _ int) bool {
		return item.Reason.Type() != billing.RatecardUsageDiscountReason
	})

	return l
}

type usageDiscountWithChildUniqueReferenceID struct {
	billing.UsageDiscount
	childUniqueReferenceID string
}

func (m *discountUsageMutator) getUsageDiscount(l PricerCalculateInput) (*usageDiscountWithChildUniqueReferenceID, error) {
	if l.line.RateCardDiscounts.Usage == nil {
		return nil, nil
	}

	rcUsageDiscount := l.line.RateCardDiscounts.Usage

	if rcUsageDiscount.CorrelationID == "" {
		return nil, fmt.Errorf("discount has no correlation ID")
	}

	return &usageDiscountWithChildUniqueReferenceID{
		UsageDiscount:          *rcUsageDiscount,
		childUniqueReferenceID: fmt.Sprintf(RateCardDiscountChildUniqueReferenceID, rcUsageDiscount.CorrelationID),
	}, nil
}

// calculateUsedQtyByCorrelationID calculates the used quantity by correlation ID for the previously billed lines
// by checking the UBP line's discounts. This works because usage discounts are presisted to the UBP line's discounts
// as they are affecting all the detailed lines.
func (m *discountUsageMutator) calculateUsedQtyByCorrelationID(l *billing.StandardLine, correlationID string) (alpacadecimal.Decimal, error) {
	if l.SplitLineHierarchy == nil {
		return alpacadecimal.Zero, errors.New("no line hierarchy is available for a progressive billed line")
	}

	usedQty := alpacadecimal.Zero

	err := l.SplitLineHierarchy.ForEachChild(billing.ForEachChildInput{
		PeriodEndLTE: l.Period.Start,
		Callback: func(child billing.LineWithInvoiceHeader) error {
			// Gathering lines do not hold usage discounts (as we don't know it yet, so they are safe to ignore)
			if child.Invoice.AsInvoice().Type() != billing.InvoiceTypeStandard {
				return nil
			}

			stdLine, err := child.Line.AsInvoiceLine().AsStandardLine()
			if err != nil {
				return err
			}

			for _, usageDiscount := range stdLine.Discounts.Usage {
				// Validate that the discount is coming from this mutator by ensuring that what we set
				// above is available here.
				if usageDiscount.Reason.Type() != billing.RatecardUsageDiscountReason {
					continue
				}

				sourceRateCardDiscount, err := usageDiscount.Reason.AsRatecardUsage()
				if err != nil {
					return fmt.Errorf("failed to convert source discount to usage discount: %w", err)
				}

				if sourceRateCardDiscount.CorrelationID == "" || sourceRateCardDiscount.CorrelationID != correlationID {
					continue
				}

				if usageDiscount.ChildUniqueReferenceID == nil ||
					*usageDiscount.ChildUniqueReferenceID != fmt.Sprintf(RateCardDiscountChildUniqueReferenceID, sourceRateCardDiscount.CorrelationID) {
					return errors.New("consistency error: the usage discount correlation ID does not match the source's correlation ID")
				}

				usedQty = usedQty.Add(usageDiscount.Quantity)
			}

			return nil
		},
	})
	if err != nil {
		return alpacadecimal.Zero, err
	}

	return usedQty, nil
}
