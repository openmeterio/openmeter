package productcatalog

import (
	"encoding/json"
	"errors"
	"fmt"

	decimal "github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/hasher"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	PercentageDiscountType DiscountType = "percentage"
	UsageDiscountType      DiscountType = "usage"
)

type DiscountType string

func (p DiscountType) Values() []DiscountType {
	return []DiscountType{
		PercentageDiscountType,
		UsageDiscountType,
	}
}

func (p DiscountType) StringValues() []string {
	return []string{
		string(PercentageDiscountType),
		string(UsageDiscountType),
	}
}

type discounter interface {
	json.Marshaler
	json.Unmarshaler

	models.Validator
	models.Clonable[Discount]
	hasher.Hasher

	Type() DiscountType
	AsPercentage() (PercentageDiscount, error)
	FromPercentage(PercentageDiscount)

	// ValidateForPrice validates the discount for a given price.
	ValidateForPrice(price *Price) error
}

var _ discounter = (*Discount)(nil)

type Discount struct {
	t          DiscountType
	percentage *PercentageDiscount
	usage      *UsageDiscount
}

func (d *Discount) Hash() hasher.Hash {
	switch d.t {
	case PercentageDiscountType:
		return d.percentage.Hash()
	case UsageDiscountType:
		return d.usage.Hash()
	default:
		return 0
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
	case UsageDiscountType:
		serde = struct {
			Type DiscountType `json:"type"`
			*UsageDiscount
		}{
			Type:          UsageDiscountType,
			UsageDiscount: d.usage,
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
	case UsageDiscountType:
		v := &UsageDiscount{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize Discount: %w", err)
		}

		d.usage = v
		d.t = UsageDiscountType
	default:
		return fmt.Errorf("invalid Discount type: %s", serde.Type)
	}

	return nil
}

func (d *Discount) Validate() error {
	switch d.t {
	case PercentageDiscountType:
		return d.percentage.Validate()
	case UsageDiscountType:
		return d.usage.Validate()
	default:
		return errors.New("invalid discount: not initialized")
	}
}

func (d *Discount) ValidateForPrice(price *Price) error {
	var errs []error

	if err := d.Validate(); err != nil {
		errs = append(errs, err)
	}

	switch d.t {
	case PercentageDiscountType:
		errs = append(errs, d.percentage.ValidateForPrice(price))
	case UsageDiscountType:
		errs = append(errs, d.usage.ValidateForPrice(price))
	default:
		errs = append(errs, errors.New("invalid discount: not initialized"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (d *Discount) Clone() Discount {
	switch d.t {
	case PercentageDiscountType:
		return NewDiscountFrom(d.percentage.Clone())
	case UsageDiscountType:
		return NewDiscountFrom(d.usage.Clone())
	default:
		return Discount{
			t: d.t,
		}
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

func (d *Discount) AsUsage() (UsageDiscount, error) {
	if d.t == "" || d.usage == nil {
		return UsageDiscount{}, errors.New("invalid discount: not initialized")
	}

	if d.t != UsageDiscountType {
		return UsageDiscount{}, fmt.Errorf("discount type mismatch: %s", d.t)
	}

	return *d.usage, nil
}

func (d *Discount) FromPercentage(discount PercentageDiscount) {
	d.percentage = &discount
	d.t = PercentageDiscountType
}

func (d *Discount) FromUsage(discount UsageDiscount) {
	d.usage = &discount
	d.t = UsageDiscountType
}

func (d Discount) Equal(other Discount) bool {
	if d.t != other.t {
		return false
	}

	switch d.t {
	case PercentageDiscountType:
		return equal.HasherPtrEqual(d.percentage, other.percentage)
	case UsageDiscountType:
		return equal.HasherPtrEqual(d.usage, other.usage)
	default:
		return false
	}
}

func NewDiscountFrom[T PercentageDiscount | UsageDiscount](v T) Discount {
	d := Discount{}

	switch any(v).(type) {
	case PercentageDiscount:
		percentage := any(v).(PercentageDiscount)
		d.FromPercentage(percentage)
	case UsageDiscount:
		usage := any(v).(UsageDiscount)
		d.FromUsage(usage)
	}

	return d
}

var (
	_ models.Validator                    = (*PercentageDiscount)(nil)
	_ hasher.Hasher                       = (*PercentageDiscount)(nil)
	_ models.Clonable[PercentageDiscount] = (*PercentageDiscount)(nil)
)

type PercentageDiscount struct {
	// Percentage defines percentage of the discount.
	Percentage models.Percentage `json:"percentage"`
}

func (f PercentageDiscount) Hash() hasher.Hash {
	var content string

	content += f.Percentage.String()

	return hasher.NewHash([]byte(content))
}

func (f PercentageDiscount) Validate() error {
	var errs []error

	if f.Percentage.LessThan(decimal.Zero) || f.Percentage.GreaterThan(decimal.NewFromInt(100)) {
		errs = append(errs, errors.New("discount percentage must be between 0 and 100"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (f PercentageDiscount) ValidateForPrice(price *Price) error {
	return nil
}

func (f PercentageDiscount) Clone() PercentageDiscount {
	return PercentageDiscount{
		Percentage: f.Percentage,
	}
}

var (
	_ models.Validator               = (*UsageDiscount)(nil)
	_ hasher.Hasher                  = (*UsageDiscount)(nil)
	_ models.Clonable[UsageDiscount] = (*UsageDiscount)(nil)
)

type UsageDiscount struct {
	Quantity decimal.Decimal `json:"quantity"`
}

func (f UsageDiscount) Hash() hasher.Hash {
	var content string

	content += f.Quantity.String()

	return hasher.NewHash([]byte(content))
}

func (f UsageDiscount) Validate() error {
	var errs []error

	if f.Quantity.LessThan(decimal.Zero) {
		errs = append(errs, errors.New("usage must be greater than 0"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (f UsageDiscount) ValidateForPrice(price *Price) error {
	var errs []error

	if price == nil {
		// We cannot validate usage discount without a price.
		return errors.New("price is required for usage discount")
	}

	if price.Type() == FlatPriceType {
		errs = append(errs, errors.New("usage discount is not supported for flat price"))
	}

	if price.Type() == DynamicPriceType {
		errs = append(errs, errors.New("usage discount is not supported for dynamic price"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (f UsageDiscount) Clone() UsageDiscount {
	return UsageDiscount{
		Quantity: f.Quantity,
	}
}

var (
	_ models.Equaler[Discounts]  = (*Discounts)(nil)
	_ models.Clonable[Discounts] = (*Discounts)(nil)
)

type Discounts []Discount

func (d Discounts) Clone() Discounts {
	// If there are no discounts, let's represent it as nil, so that testing is easier
	if len(d) == 0 {
		return nil
	}

	clone := make(Discounts, len(d))

	for i, discount := range d {
		clone[i] = discount.Clone()
	}

	return clone
}

func (d Discounts) Equal(v Discounts) bool {
	if len(d) != len(v) {
		return false
	}

	leftSet := make(map[hasher.Hash]struct{})
	for _, discount := range d {
		leftSet[discount.Hash()] = struct{}{}
	}

	rightSet := make(map[hasher.Hash]struct{})
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

func (d Discounts) ValidateForPrice(price *Price) error {
	var errs []error

	if len(d) == 0 {
		return nil
	}

	sumPercentage := models.Percentage{}

	for i, discount := range d {
		if err := discount.ValidateForPrice(price); err != nil {
			errs = append(errs, fmt.Errorf("discounts[%d]: %w", i, err))
		}

		if discount.Type() == PercentageDiscountType {
			percentage, err := discount.AsPercentage()
			if err != nil {
				errs = append(errs, fmt.Errorf("discounts[%d]: %w", i, err))
			}

			sumPercentage = sumPercentage.Add(percentage.Percentage)
		}
	}

	if sumPercentage.GreaterThan(decimal.NewFromInt(100)) {
		errs = append(errs, errors.New("sum of percentage discounts cannot be greater than 100"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
