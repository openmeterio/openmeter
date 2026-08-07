package detailedline

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Base is the common model for charge realization detailed lines. Billing
// invoice detailed lines cannot share this model because billing discounts are
// managed resources, while charge discounts are signed realization snapshots.
type Base struct {
	stddetailedline.Base

	AmountDiscounts AmountDiscounts `json:"amountDiscounts,omitempty"`
}

func (l Base) Clone() Base {
	l.Base = l.Base.Clone()
	l.AmountDiscounts = l.AmountDiscounts.Clone()

	return l
}

// CloneStandardBase returns the shared detailed-line fields without charge
// discount snapshots. Billing manages its own discount resources when it
// consumes the returned base.
func (l Base) CloneStandardBase() stddetailedline.Base {
	return l.Base.Clone()
}

func (l Base) Validate(opts ...stddetailedline.ValidateOption) error {
	var errs []error

	if err := l.Base.Validate(opts...); err != nil {
		errs = append(errs, err)
	}

	if err := l.AmountDiscounts.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("amount discounts: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

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

// MapAmountDiscountsFromBilling captures billing rating discounts as charge
// realization facts. Managed billing identifiers are intentionally excluded;
// the child reference is the stable identity used across rerating and
// correction runs.
func MapAmountDiscountsFromBilling(discounts billing.AmountLineDiscountsManaged) AmountDiscounts {
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
