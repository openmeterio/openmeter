package productcatalog

import (
	"errors"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/hasher"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_ models.Validator                           = (*PercentageDiscount)(nil)
	_ hasher.Hasher                              = (*PercentageDiscount)(nil)
	_ models.Clonable[PercentageDiscount]        = (*PercentageDiscount)(nil)
	_ models.CustomValidator[PercentageDiscount] = (*PercentageDiscount)(nil)
)

type PercentageDiscount struct {
	// Percentage defines percentage of the discount.
	Percentage models.Percentage `json:"percentage"`
}

func (d PercentageDiscount) Hash() hasher.Hash {
	var content string

	content += d.Percentage.String()

	return hasher.NewHash([]byte(content))
}

func (d PercentageDiscount) ValidateWith(v ...models.ValidatorFunc[PercentageDiscount]) error {
	return models.Validate(d, v...)
}

func PercentageDiscountWithValidValue() models.ValidatorFunc[PercentageDiscount] {
	return func(d PercentageDiscount) error {
		if d.Percentage.LessThan(decimal.Zero) || d.Percentage.GreaterThan(decimal.NewFromInt(100)) {
			return ErrPercentageDiscountInvalidValue
		}

		return nil
	}
}

func (d PercentageDiscount) Validate() error {
	return d.ValidateWith(PercentageDiscountWithValidValue())
}

func (d PercentageDiscount) ValidateForPrice(_ *Price) error {
	return d.Validate()
}

func (d PercentageDiscount) Clone() PercentageDiscount {
	return PercentageDiscount{
		Percentage: d.Percentage,
	}
}

var (
	_ models.Validator                      = (*UsageDiscount)(nil)
	_ hasher.Hasher                         = (*UsageDiscount)(nil)
	_ models.Clonable[UsageDiscount]        = (*UsageDiscount)(nil)
	_ models.CustomValidator[UsageDiscount] = (*UsageDiscount)(nil)
)

type UsageDiscount struct {
	Quantity decimal.Decimal `json:"quantity"`
}

func (d UsageDiscount) Hash() hasher.Hash {
	var content string

	content += d.Quantity.String()

	return hasher.NewHash([]byte(content))
}

func (d UsageDiscount) ValidateWith(v ...models.ValidatorFunc[UsageDiscount]) error {
	return models.Validate(d, v...)
}

func UsageDiscountWithValidValue() models.ValidatorFunc[UsageDiscount] {
	return func(d UsageDiscount) error {
		if d.Quantity.LessThan(decimal.Zero) {
			return ErrUsageDiscountNegativeQuantity
		}

		return nil
	}
}

func UsageDiscountWithPrice(price *Price) models.ValidatorFunc[UsageDiscount] {
	return func(d UsageDiscount) error {
		var errs []error

		if price == nil {
			// We cannot validate usage discount without a price.
			return errors.New("price is required for usage discount")
		}

		if err := d.Validate(); err != nil {
			errs = append(errs, err)
		}

		if price.Type() == FlatPriceType {
			errs = append(errs, ErrUsageDiscountWithFlatPrice)
		}

		return errors.Join(errs...)
	}
}

func (d UsageDiscount) Validate() error {
	return d.ValidateWith(UsageDiscountWithValidValue())
}

func (d UsageDiscount) ValidateForPrice(price *Price) error {
	return d.ValidateWith(UsageDiscountWithPrice(price))
}

func (d UsageDiscount) Clone() UsageDiscount {
	return UsageDiscount{
		Quantity: d.Quantity,
	}
}

var (
	_ models.Equaler[Discounts]  = (*Discounts)(nil)
	_ models.Clonable[Discounts] = (*Discounts)(nil)
	_ models.Validator           = (*Discounts)(nil)
)

type Discounts struct {
	Percentage *PercentageDiscount `json:"percentage,omitempty"`
	Usage      *UsageDiscount      `json:"usage,omitempty"`
}

func (d Discounts) Equal(v Discounts) bool {
	if !equal.HasherPtrEqual(d.Percentage, v.Percentage) {
		return false
	}

	if !equal.HasherPtrEqual(d.Usage, v.Usage) {
		return false
	}

	return true
}

func (d Discounts) Clone() Discounts {
	out := Discounts{}

	if d.Percentage != nil {
		out.Percentage = lo.ToPtr(d.Percentage.Clone())
	}

	if d.Usage != nil {
		out.Usage = lo.ToPtr(d.Usage.Clone())
	}

	return out
}

func (d *Discounts) Validate() error {
	var errs []error

	if d == nil {
		return nil
	}

	if d.Percentage != nil {
		if err := d.Percentage.Validate(); err != nil {
			errs = append(errs, models.ErrorWithFieldPrefix(
				models.NewFieldSelectorGroup(models.NewFieldSelector("percentage")),
				err),
			)
		}
	}

	if d.Usage != nil {
		if err := d.Usage.Validate(); err != nil {
			errs = append(errs, models.ErrorWithFieldPrefix(
				models.NewFieldSelectorGroup(models.NewFieldSelector("usage")),
				err),
			)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return models.NewGenericValidationError(models.ErrorWithFieldPrefix(
			models.NewFieldSelectorGroup(models.NewFieldSelector("discounts")),
			err),
		)
	}

	return nil
}

func (d Discounts) ValidateForPrice(price *Price) error {
	var errs []error

	if !d.IsEmpty() && price == nil {
		return errors.New("price is required for discounts")
	}

	if d.Percentage != nil {
		if err := d.Percentage.ValidateForPrice(price); err != nil {
			errs = append(errs, models.ErrorWithFieldPrefix(
				models.NewFieldSelectorGroup(models.NewFieldSelector("percentage")),
				err),
			)
		}
	}

	if d.Usage != nil {
		if err := d.Usage.ValidateForPrice(price); err != nil {
			errs = append(errs, models.ErrorWithFieldPrefix(
				models.NewFieldSelectorGroup(models.NewFieldSelector("usage")),
				err),
			)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return models.NewGenericValidationError(models.ErrorWithFieldPrefix(
			models.NewFieldSelectorGroup(models.NewFieldSelector("discounts")),
			err),
		)
	}

	return nil
}

func (d Discounts) IsEmpty() bool {
	return lo.IsEmpty(d)
}
