package billing

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

type discountType[T any] interface {
	models.Clonable[T]
	models.Equaler[T]
	models.Validator
}

// Extended discount types

var _ discountType[PercentageDiscount] = (*PercentageDiscount)(nil)

type PercentageDiscount struct {
	productcatalog.PercentageDiscount `json:",inline"`

	CorrelationID string `json:"correlationID"`
}

func (d PercentageDiscount) Clone() PercentageDiscount {
	return PercentageDiscount{
		PercentageDiscount: d.PercentageDiscount.Clone(),
		CorrelationID:      d.CorrelationID,
	}
}

func (d PercentageDiscount) Equal(other PercentageDiscount) bool {
	if d.PercentageDiscount.Hash() != other.PercentageDiscount.Hash() {
		return false
	}

	if d.CorrelationID != other.CorrelationID {
		return false
	}

	return true
}

type UsageDiscount struct {
	productcatalog.UsageDiscount `json:",inline"`

	CorrelationID string `json:"correlationID"`
}

var _ discountType[UsageDiscount] = (*UsageDiscount)(nil)

func (d UsageDiscount) Clone() UsageDiscount {
	return UsageDiscount{
		UsageDiscount: d.UsageDiscount.Clone(),
		CorrelationID: d.CorrelationID,
	}
}

func (d UsageDiscount) Equal(other UsageDiscount) bool {
	if d.UsageDiscount.Hash() != other.UsageDiscount.Hash() {
		return false
	}

	if d.CorrelationID != other.CorrelationID {
		return false
	}

	return true
}

// Discount type
type discounter interface {
	json.Marshaler
	json.Unmarshaler

	models.Clonable[Discount]
	models.Equaler[Discount]
	models.Validator

	Type() productcatalog.DiscountType
	AsPercentage() (PercentageDiscount, error)
	AsUsage() (UsageDiscount, error)

	ValidateForPrice(price *Price) error
}

var _ discounter = (*Discount)(nil)

type Discount struct {
	t productcatalog.DiscountType

	percentage *PercentageDiscount
	usage      *UsageDiscount
}

func NewDiscountFrom[T PercentageDiscount | UsageDiscount | productcatalog.PercentageDiscount | productcatalog.UsageDiscount](in T) Discount {
	switch d := any(in).(type) {
	case PercentageDiscount:
		percentage := any(d).(PercentageDiscount)
		return Discount{
			t:          productcatalog.PercentageDiscountType,
			percentage: &percentage,
		}
	case productcatalog.PercentageDiscount:
		percentage := any(d).(productcatalog.PercentageDiscount)
		return Discount{
			t: productcatalog.PercentageDiscountType,
			percentage: &PercentageDiscount{
				PercentageDiscount: percentage,
			},
		}
	case UsageDiscount:
		usage := any(d).(UsageDiscount)
		return Discount{
			t:     productcatalog.UsageDiscountType,
			usage: &usage,
		}
	case productcatalog.UsageDiscount:
		usage := any(d).(productcatalog.UsageDiscount)
		return Discount{
			t: productcatalog.UsageDiscountType,
			usage: &UsageDiscount{
				UsageDiscount: usage,
			},
		}
	}

	return Discount{}
}

func (d *Discount) MarshalJSON() ([]byte, error) {
	var serde interface{}

	switch d.t {
	case productcatalog.PercentageDiscountType:
		serde = struct {
			Type productcatalog.DiscountType `json:"type"`
			*PercentageDiscount
		}{
			Type:               productcatalog.PercentageDiscountType,
			PercentageDiscount: d.percentage,
		}
	case productcatalog.UsageDiscountType:
		serde = struct {
			Type productcatalog.DiscountType `json:"type"`
			*UsageDiscount
		}{
			Type:          productcatalog.UsageDiscountType,
			UsageDiscount: d.usage,
		}
	default:
		return nil, fmt.Errorf("invalid Discount type: %s", d.t)
	}

	b, err := json.Marshal(serde)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON serialize Discount: %w", err)
	}

	return b, nil
}

func (d *Discount) UnmarshalJSON(bytes []byte) error {
	serde := &struct {
		Type productcatalog.DiscountType `json:"type"`
	}{}

	if err := json.Unmarshal(bytes, serde); err != nil {
		return fmt.Errorf("failed to JSON deserialize Discount type: %w", err)
	}

	switch serde.Type {
	case productcatalog.PercentageDiscountType:
		v := &PercentageDiscount{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize Discount: %w", err)
		}

		d.percentage = v
		d.t = productcatalog.PercentageDiscountType
	case productcatalog.UsageDiscountType:
		v := &UsageDiscount{}
		if err := json.Unmarshal(bytes, v); err != nil {
			return fmt.Errorf("failed to JSON deserialize Discount: %w", err)
		}

		d.usage = v
		d.t = productcatalog.UsageDiscountType
	default:
		return fmt.Errorf("invalid Discount type: %s", serde.Type)
	}

	return nil
}

func (d *Discount) Clone() Discount {
	switch d.t {
	case productcatalog.PercentageDiscountType:
		return Discount{
			t:          d.t,
			percentage: lo.ToPtr(d.percentage.Clone()),
		}
	case productcatalog.UsageDiscountType:
		return Discount{
			t:     d.t,
			usage: lo.ToPtr(d.usage.Clone()),
		}
	}

	return Discount{}
}

func (d *Discount) Equal(other Discount) bool {
	if d.t != other.t {
		return false
	}

	switch d.t {
	case productcatalog.PercentageDiscountType:
		return d.percentage.Equal(*other.percentage)
	case productcatalog.UsageDiscountType:
		return d.usage.Equal(*other.usage)
	}

	return false
}

func (d *Discount) Validate() error {
	switch d.t {
	case productcatalog.PercentageDiscountType:
		return d.percentage.Validate()
	case productcatalog.UsageDiscountType:
		return d.usage.Validate()
	}

	return errors.New("invalid discount type")
}

func (d *Discount) ValidateForPrice(price *productcatalog.Price) error {
	switch d.t {
	case productcatalog.PercentageDiscountType:
		return d.percentage.ValidateForPrice(price)
	case productcatalog.UsageDiscountType:
		return d.usage.ValidateForPrice(price)
	}

	return errors.New("invalid discount type")
}

func (d *Discount) Type() productcatalog.DiscountType {
	return d.t
}

func (d *Discount) AsPercentage() (PercentageDiscount, error) {
	if d.t != productcatalog.PercentageDiscountType {
		return PercentageDiscount{}, errors.New("invalid discount type")
	}

	if d.percentage == nil {
		return PercentageDiscount{}, errors.New("percentage discount is missing")
	}

	return *d.percentage, nil
}

func (d *Discount) AsUsage() (UsageDiscount, error) {
	if d.t != productcatalog.UsageDiscountType {
		return UsageDiscount{}, errors.New("invalid discount type")
	}

	if d.usage == nil {
		return UsageDiscount{}, errors.New("usage discount is missing")
	}

	return *d.usage, nil
}

func (d Discount) WithCorrelationID(correlationID string) Discount {
	res := d.Clone()

	switch d.t {
	case productcatalog.PercentageDiscountType:
		res.percentage.CorrelationID = correlationID
	case productcatalog.UsageDiscountType:
		res.usage.CorrelationID = correlationID
	}

	return res
}

type Discounts []Discount

func (d Discounts) Clone() Discounts {
	if len(d) == 0 {
		return nil
	}

	out := make(Discounts, len(d))
	for i, discount := range d {
		out[i] = discount.Clone()
	}
	return out
}

func (d Discounts) ValidateForPrice(price *productcatalog.Price) error {
	var errs []error

	for _, discount := range d {
		if err := discount.ValidateForPrice(price); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
