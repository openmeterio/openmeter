package plan

import (
	"encoding/json"
	"errors"
	"fmt"

	decimal "github.com/alpacahq/alpacadecimal"
)

const (
	PercentageDiscountType DiscountType = "percentage"
)

type DiscountType string

func (p DiscountType) Values() []DiscountType {
	return []DiscountType{
		PercentageDiscountType,
	}
}

func (p DiscountType) StringValues() []string {
	return []string{
		string(PercentageDiscountType),
	}
}

type discounter interface {
	json.Marshaler
	json.Unmarshaler
	Validator

	Type() DiscountType
	AsPercentage() (PercentageDiscount, error)
	FromPercentage(PercentageDiscount)
}

var _ discounter = (*Discount)(nil)

type Discount struct {
	t          DiscountType
	percentage *PercentageDiscount
}

func (d *Discount) MarshalJSON() ([]byte, error) {
	var b []byte
	var err error

	switch d.t {
	case PercentageDiscountType:
		b, err = json.Marshal(d.percentage)
		if err != nil {
			return nil, fmt.Errorf("failed to json marshal percentage discount: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid discount type: %s", d.t)
	}

	return b, nil
}

func (d *Discount) UnmarshalJSON(bytes []byte) error {
	meta := &DiscountMeta{}

	if err := json.Unmarshal(bytes, meta); err != nil {
		return fmt.Errorf("failed to json unmarshal discount type: %w", err)
	}

	switch meta.Type {
	case PercentageDiscountType:
		v := &PercentageDiscount{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to json unmarshal percentage discount: %w", err)
		}

		d.percentage = v
		d.t = PercentageDiscountType
	default:
		return fmt.Errorf("invalid discount type: %s", meta.Type)
	}

	return nil
}

func (d *Discount) Validate() error {
	switch d.t {
	case PercentageDiscountType:
		return d.percentage.Validate()
	default:
		return errors.New("invalid discount: not initialized")
	}
}

func (d *Discount) Type() DiscountType {
	return d.t
}

func (d *Discount) AsPercentage() (PercentageDiscount, error) {
	if d.t == "" || d.percentage == nil {
		return PercentageDiscount{}, errors.New("invalid discount: not initialized")
	}

	if d.t != PercentageDiscountType {
		return PercentageDiscount{}, fmt.Errorf("discount type mismatch: %s", d.t)
	}

	return *d.percentage, nil
}

func (d *Discount) FromPercentage(discount PercentageDiscount) {
	d.percentage = &discount
	d.t = PercentageDiscountType
}

func NewDiscountFrom[T PercentageDiscount](v T) Discount {
	d := Discount{}

	switch any(v).(type) {
	case FlatPrice:
		percentage := any(v).(PercentageDiscount)
		d.FromPercentage(percentage)
	}

	return d
}

type DiscountMeta struct {
	// Type of the Discount.
	Type DiscountType `json:"type"`
}

var _ Validator = (*PercentageDiscount)(nil)

type PercentageDiscount struct {
	DiscountMeta

	// Percentage defines percentage of the discount.
	Percentage decimal.Decimal `json:"percentage"`
}

func (f PercentageDiscount) Validate() error {
	var errs []error

	if f.Percentage.LessThan(decimal.Zero) || f.Percentage.GreaterThan(decimal.NewFromInt(100)) {
		errs = append(errs, errors.New("discount percentage must be between 0 and 100"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
