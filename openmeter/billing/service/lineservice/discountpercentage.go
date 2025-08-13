package lineservice

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type discountPercentageMutator struct{}

var _ PostCalculationMutator = (*discountPercentageMutator)(nil)

func (m *discountPercentageMutator) Mutate(i PricerCalculateInput, pricerResult newDetailedLinesInput) (newDetailedLinesInput, error) {
	discount, err := m.getDiscount(i.line.RateCardDiscounts)
	if err != nil {
		return nil, err
	}

	if discount == nil {
		return pricerResult, nil
	}

	currencyCalc, err := i.Currency().Calculator()
	if err != nil {
		return nil, err
	}

	out, err := slicesx.MapWithErr(pricerResult, func(l newDetailedLineInput) (newDetailedLineInput, error) {
		lineDiscount, err := m.getLineDiscount(l.TotalAmount(currencyCalc), currencyCalc, *discount)
		if err != nil {
			return newDetailedLineInput{}, err
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

func (m *discountPercentageMutator) getDiscount(discounts billing.Discounts) (*discountWithChildReferenceID, error) {
	if discounts.Percentage == nil {
		return nil, nil
	}

	if discounts.Percentage.CorrelationID == "" {
		return nil, errors.New("correlation ID is required for rate card discounts")
	}

	return &discountWithChildReferenceID{
		PercentageDiscount:     *discounts.Percentage,
		ChildUniqueReferenceID: fmt.Sprintf(RateCardDiscountChildUniqueReferenceID, discounts.Percentage.CorrelationID),
	}, nil
}

func (m *discountPercentageMutator) getLineDiscount(lineTotal alpacadecimal.Decimal, currency currencyx.Calculator, discount discountWithChildReferenceID) (billing.AmountLineDiscountManaged, error) {
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
