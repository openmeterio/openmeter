package mutator

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type DiscountPercentage struct{}

var _ PostCalculationMutator = (*DiscountPercentage)(nil)

func (m *DiscountPercentage) Mutate(in rate.PricerCalculateInput, pricerResult rating.DetailedLines) (rating.DetailedLines, error) {
	discount, err := m.getDiscount(in.GetRateCardDiscounts())
	if err != nil {
		return nil, err
	}

	if discount == nil {
		return pricerResult, nil
	}

	out, err := slicesx.MapWithErr(pricerResult, func(l rating.DetailedLine) (rating.DetailedLine, error) {
		lineDiscount, err := m.getLineDiscount(l.TotalAmount(in.CurrencyCalculator), in.CurrencyCalculator, *discount)
		if err != nil {
			return rating.DetailedLine{}, err
		}

		l.AmountDiscounts = append(l.AmountDiscounts, lineDiscount)
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

func (m *DiscountPercentage) getDiscount(discounts billing.Discounts) (*discountWithChildReferenceID, error) {
	if discounts.Percentage == nil {
		return nil, nil
	}

	if discounts.Percentage.CorrelationID == "" {
		return nil, errors.New("correlation ID is required for rate card discounts")
	}

	return &discountWithChildReferenceID{
		PercentageDiscount:     *discounts.Percentage,
		ChildUniqueReferenceID: fmt.Sprintf(rating.RateCardDiscountChildUniqueReferenceID, discounts.Percentage.CorrelationID),
	}, nil
}

func (m *DiscountPercentage) getLineDiscount(lineTotal alpacadecimal.Decimal, currency currencyx.Calculator, discount discountWithChildReferenceID) (billing.AmountLineDiscountManaged, error) {
	if discount.Percentage.GreaterThan(alpacadecimal.NewFromInt(100)) || discount.Percentage.LessThan(alpacadecimal.Zero) {
		return billing.AmountLineDiscountManaged{}, errors.New("total discount percentage cannot be greater than 100 or less than 0")
	}

	return billing.AmountLineDiscountManaged{
		AmountLineDiscount: billing.AmountLineDiscount{
			LineDiscountBase: billing.LineDiscountBase{
				ChildUniqueReferenceID: &discount.ChildUniqueReferenceID,
				Reason:                 billing.NewDiscountReasonFrom(discount.PercentageDiscount),
			},
			Amount: currency.RoundToPrecision(discount.Percentage.ApplyTo(lineTotal)),
		},
	}, nil
}
