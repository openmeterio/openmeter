package amountdiscount

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// AmountDiscount is a signed discount realization on a charge detailed line.
// Negative values represent the reversal of a previously realized discount.
type AmountDiscount struct {
	ChildUniqueReferenceID string                 `json:"childUniqueReferenceID"`
	Description            *string                `json:"description,omitempty"`
	Reason                 billing.DiscountReason `json:"reason"`
	Amount                 alpacadecimal.Decimal  `json:"amount"`
	RoundingAmount         alpacadecimal.Decimal  `json:"roundingAmount"`
}

func (d AmountDiscount) Clone() AmountDiscount {
	if d.Description != nil {
		d.Description = lo.ToPtr(*d.Description)
	}

	return d
}

func (d AmountDiscount) Validate() error {
	var errs []error

	if d.ChildUniqueReferenceID == "" {
		errs = append(errs, errors.New("child unique reference id is required"))
	}

	if err := d.Reason.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("reason: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type AmountDiscounts []AmountDiscount

// New captures billing rating discounts as charge realization facts. Managed
// billing identifiers are intentionally excluded; the child reference is the
// stable identity used across rerating and correction runs.
func New(discounts billing.AmountLineDiscountsManaged) AmountDiscounts {
	return lo.Map(discounts, func(discount billing.AmountLineDiscountManaged, _ int) AmountDiscount {
		return AmountDiscount{
			ChildUniqueReferenceID: lo.FromPtr(discount.ChildUniqueReferenceID),
			Description:            discount.Description,
			Reason:                 discount.Reason,
			Amount:                 discount.Amount,
			RoundingAmount:         discount.RoundingAmount,
		}
	})
}

// AmountDiscountsOption is the named persistence representation of optional
// amount discounts. Ent cannot generate code for parameterized GoType values.
type AmountDiscountsOption struct {
	mo.Option[AmountDiscounts]
}

func NewAmountDiscountsOption(option mo.Option[AmountDiscounts]) AmountDiscountsOption {
	return AmountDiscountsOption{Option: option}
}

func (d AmountDiscounts) Clone() AmountDiscounts {
	if d == nil {
		return nil
	}

	out := make(AmountDiscounts, len(d))
	for idx, discount := range d {
		out[idx] = discount.Clone()
	}

	return out
}

func (d AmountDiscounts) Validate() error {
	var errs []error

	for idx, discount := range d {
		if err := discount.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("[%d]: %w", idx, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (d AmountDiscounts) SumAmount(currency currencyx.Currency) alpacadecimal.Decimal {
	sum := alpacadecimal.Zero

	for _, discount := range d {
		sum = sum.
			Add(currency.RoundToPrecision(discount.Amount)).
			Add(currency.RoundToPrecision(discount.RoundingAmount))
	}

	return sum
}
