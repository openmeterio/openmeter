package lineservice

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type discountPercentageMutator struct{}

var _ PostCalculationMutator = (*discountPercentageMutator)(nil)

func (m *discountPercentageMutator) Mutate(i PricerCalculateInput, pricerResult newDetailedLinesInput) (newDetailedLinesInput, error) {
	discounts, err := m.getDiscounts(i.line.RateCardDiscounts)
	if err != nil {
		return nil, err
	}

	if len(discounts) == 0 {
		return pricerResult, nil
	}

	currencyCalc, err := i.Currency().Calculator()
	if err != nil {
		return nil, err
	}

	out, err := slicesx.MapWithErr(pricerResult, func(l newDetailedLineInput) (newDetailedLineInput, error) {
		lineDiscounts, err := m.getLineDiscounts(l.TotalAmount(currencyCalc), currencyCalc, discounts)
		if err != nil {
			return newDetailedLineInput{}, err
		}

		l.Discounts = append(l.Discounts, lineDiscounts...)
		return l, nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

type discountWithChildReferenceID struct {
	billing.PercentageDiscount
	ChildUniqueReferenceID string
}

func (m *discountPercentageMutator) getDiscounts(discounts billing.Discounts) ([]discountWithChildReferenceID, error) {
	percentageDiscounts := []discountWithChildReferenceID{}
	for _, discount := range discounts {
		if discount.Type() != productcatalog.PercentageDiscountType {
			continue
		}

		percentage, err := discount.AsPercentage()
		if err != nil {
			return nil, err
		}

		if percentage.CorrelationID == "" {
			return nil, errors.New("correlation ID is required for rate card discounts")
		}

		percentageDiscounts = append(percentageDiscounts, discountWithChildReferenceID{
			PercentageDiscount:     percentage,
			ChildUniqueReferenceID: fmt.Sprintf(RateCardDiscountChildUniqueReferenceID, percentage.CorrelationID),
		})
	}

	return percentageDiscounts, nil
}

func (m *discountPercentageMutator) getLineDiscounts(lineTotal alpacadecimal.Decimal, currency currencyx.Calculator, discounts []discountWithChildReferenceID) (billing.LineDiscounts, error) {
	totalDiscount := models.Percentage{}

	for _, discount := range discounts {
		totalDiscount = totalDiscount.Add(discount.Percentage)
	}

	if totalDiscount.GreaterThan(alpacadecimal.NewFromInt(100)) {
		return nil, errors.New("total discount percentage cannot be greater than 100")
	}

	lineDiscounts := []billing.AmountLineDiscount{}
	for _, discount := range discounts {
		amount := currency.RoundToPrecision(discount.Percentage.ApplyTo(lineTotal))

		lineDiscounts = append(lineDiscounts, billing.AmountLineDiscount{
			LineDiscountBase: billing.LineDiscountBase{
				ChildUniqueReferenceID: &discount.ChildUniqueReferenceID,
				Reason:                 billing.LineDiscountReasonRatecardDiscount,
				SourceDiscount:         lo.ToPtr(billing.NewDiscountFrom(discount.PercentageDiscount)),
			},
			Amount: amount,
		})
	}

	sumOfDiscounts := alpacadecimal.Zero
	for _, discount := range lineDiscounts {
		sumOfDiscounts = sumOfDiscounts.Add(discount.Amount)
	}

	totalDiscountAmount := currency.RoundToPrecision(totalDiscount.ApplyTo(lineTotal))
	// Rounding support (e.g. a 100% of discount composed of 33%, 33%, 34% of 0.01$ should yield 0.0$)
	roundingAmount := totalDiscountAmount.Sub(sumOfDiscounts)

	if !roundingAmount.IsZero() {
		for i := len(lineDiscounts) - 1; i >= 0; i-- {
			discount := lineDiscounts[i]

			if discount.Amount.Add(roundingAmount).GreaterThanOrEqual(alpacadecimal.Zero) {
				discount.RoundingAmount = roundingAmount
				lineDiscounts[i] = discount
				break
			}
		}
	}

	out := billing.LineDiscounts{}
	for _, discount := range lineDiscounts {
		out = append(out, billing.NewLineDiscountFrom(discount))
	}

	return out, nil
}
