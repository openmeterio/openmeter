package productcatalog

import (
	"encoding/json"
	"errors"
	"fmt"

	decimal "github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/hasher"
	"github.com/openmeterio/openmeter/pkg/models"
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

	models.Validator
	hasher.Hasher

	Type() DiscountType
	RateCardKeys() []string
	AsPercentage() (PercentageDiscount, error)
	FromPercentage(PercentageDiscount)
}

var _ discounter = (*Discount)(nil)

type Discount struct {
	t          DiscountType
	percentage *PercentageDiscount
}

func (d *Discount) Hash() hasher.Hash {
	switch d.t {
	case PercentageDiscountType:
		return d.percentage.Hash()
	default:
		return 0
	}
}

func (d *Discount) RateCardKeys() []string {
	switch d.t {
	case PercentageDiscountType:
		return d.percentage.RateCards
	default:
		return nil
	}
}

func (d *Discount) MarshalJSON() ([]byte, error) {
	var b []byte
	var err error
	var serde interface{}

	switch d.t {
	case PercentageDiscountType:
		serde = struct {
			Type DiscountType `json:"type"`
			*PercentageDiscount
		}{
			Type:               PercentageDiscountType,
			PercentageDiscount: d.percentage,
		}
	default:
		return nil, fmt.Errorf("invalid Discount type: %s", d.t)
	}

	b, err = json.Marshal(serde)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON serialize Discount: %w", err)
	}

	return b, nil
}

func (d *Discount) UnmarshalJSON(bytes []byte) error {
	serde := &struct {
		Type DiscountType `json:"type"`
	}{}

	if err := json.Unmarshal(bytes, serde); err != nil {
		return fmt.Errorf("failed to JSON deserialize Discount type: %w", err)
	}

	switch serde.Type {
	case PercentageDiscountType:
		v := &PercentageDiscount{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize Discount: %w", err)
		}

		d.percentage = v
		d.t = PercentageDiscountType
	default:
		return fmt.Errorf("invalid Discount type: %s", serde.Type)
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
	case PercentageDiscount:
		percentage := any(v).(PercentageDiscount)
		d.FromPercentage(percentage)
	}

	return d
}

var (
	_ models.Validator = (*PercentageDiscount)(nil)
	_ hasher.Hasher    = (*PercentageDiscount)(nil)
)

type PercentageDiscount struct {
	// Percentage defines percentage of the discount.
	Percentage decimal.Decimal `json:"percentage"`

	// RateCards is the list of specific RateCard Keys the discount is applied to.
	// If not provided the discount applies to all RateCards in Phase.
	RateCards []string `json:"rateCards,omitempty"`
}

func (f PercentageDiscount) Hash() hasher.Hash {
	var content string

	content += f.Percentage.String()

	for _, rateCardName := range f.RateCards {
		content += rateCardName
	}

	return hasher.NewHash([]byte(content))
}

func (f PercentageDiscount) Validate() error {
	var errs []error

	if f.Percentage.LessThan(decimal.Zero) || f.Percentage.GreaterThan(decimal.NewFromInt(100)) {
		errs = append(errs, errors.New("discount percentage must be between 0 and 100"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ models.Equaler[Discounts] = (*Discounts)(nil)

type Discounts []Discount

func (d Discounts) Equal(v Discounts) bool {
	if len(d) != len(v) {
		return false
	}

	leftSet := make(map[uint64]struct{})
	for _, discount := range d {
		leftSet[discount.Hash()] = struct{}{}
	}

	rightSet := make(map[uint64]struct{})
	for _, discount := range v {
		rightSet[discount.Hash()] = struct{}{}
	}

	if len(leftSet) != len(rightSet) {
		return false
	}

	var visited int
	for key, left := range leftSet {
		right, ok := rightSet[key]
		if !ok {
			return false
		}

		if left != right {
			return false
		}

		visited++
	}

	return visited == len(rightSet)
}
