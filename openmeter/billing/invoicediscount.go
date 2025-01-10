package billing

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/pkg/models"
)

type InvoiceDiscountType string

const (
	PercentageDiscountType InvoiceDiscountType = "percentage"
	// TODO[OM-1076]: implement amount discount
	// AmountDiscountType     InvoiceDiscountType = "amount"
)

func (d InvoiceDiscountType) Values() []string {
	return []string{string(PercentageDiscountType)}
}

type InvoiceDiscountID models.NamespacedID

type InvoiceDiscountBase struct {
	models.ManagedResource

	InvoiceID string              `json:"invoice_id"`
	Type      InvoiceDiscountType `json:"type"`
	LineIDs   []string            `json:"line_ids"`
}

func (d *InvoiceDiscountBase) Validate() error {
	var err error

	if d.InvoiceID == "" {
		err = errors.Join(err, errors.New("invoice ID is required"))
	}

	if d.Type == "" || !lo.Contains(d.Type.Values(), string(d.Type)) {
		err = errors.Join(err, errors.New("invalid discount type"))
	}

	return err
}

func (d InvoiceDiscountBase) Equals(other InvoiceDiscountBase) bool {
	// TODO[later]: Use hashing instead of reflection if we have performance issues
	return reflect.DeepEqual(d, other)
}

func (d InvoiceDiscountBase) DiscountID() InvoiceDiscountID {
	return InvoiceDiscountID{
		Namespace: d.Namespace,
		ID:        d.ID,
	}
}

type InvoiceDiscountPercentage struct {
	InvoiceDiscountBase

	Percentage alpacadecimal.Decimal `json:"percentage"`
}

func (d *InvoiceDiscountPercentage) Validate() error {
	err := d.InvoiceDiscountBase.Validate()

	if d.Percentage.LessThan(alpacadecimal.Zero) || d.Percentage.GreaterThan(alpacadecimal.NewFromInt(100)) {
		err = errors.Join(err, errors.New("discount percentage must be between 0 and 100"))
	}

	return err
}

func (d *InvoiceDiscountPercentage) Equals(other InvoiceDiscountPercentage) bool {
	return d.Percentage.Equal(other.Percentage) || d.InvoiceDiscountBase.Equals(other.InvoiceDiscountBase)
}

func (d InvoiceDiscountPercentage) Clone() *InvoiceDiscountPercentage {
	clone := &d

	clone.LineIDs = make([]string, 0, len(d.LineIDs))
	copy(clone.LineIDs, d.LineIDs)

	return clone
}

type invoiceDiscount interface {
	json.Marshaler
	json.Unmarshaler
	models.Validator

	Type() InvoiceDiscountType
	AsPercentage() (InvoiceDiscountPercentage, error)
	FromPercentage(InvoiceDiscountPercentage)
	DiscountBase() (InvoiceDiscountBase, error)

	Equals(InvoiceDiscount) bool
}

var _ invoiceDiscount = (*InvoiceDiscount)(nil)

type InvoiceDiscount struct {
	t          InvoiceDiscountType
	percentage *InvoiceDiscountPercentage
}

func NewInvoiceDiscountFrom[T InvoiceDiscountPercentage](v T) InvoiceDiscount {
	discount := InvoiceDiscount{}

	switch any(v).(type) {
	case InvoiceDiscountPercentage:
		pct := any(v).(InvoiceDiscountPercentage)
		discount.FromPercentage(pct)
	}

	return discount
}

func (d *InvoiceDiscount) FromPercentage(v InvoiceDiscountPercentage) {
	d.t = v.Type
	d.percentage = &v
}

func (d *InvoiceDiscount) AsPercentage() (InvoiceDiscountPercentage, error) {
	if d.t == "" || d.percentage == nil {
		return InvoiceDiscountPercentage{}, fmt.Errorf("discount is not initialized")
	}

	if d.t != PercentageDiscountType {
		return InvoiceDiscountPercentage{}, fmt.Errorf("discount is not a percentage discount")
	}

	return *d.percentage, nil
}

func (d *InvoiceDiscount) MarshalJSON() ([]byte, error) {
	var serde interface{}

	switch d.t {
	case PercentageDiscountType:
		serde = &struct {
			Type InvoiceDiscountType `json:"type"`
			*InvoiceDiscountPercentage
		}{
			Type:                      d.t,
			InvoiceDiscountPercentage: d.percentage,
		}
	default:
		return nil, fmt.Errorf("unsupported discount type %s", d.t)
	}

	b, err := json.Marshal(serde)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON serialize invoice discount: %w", err)
	}

	return b, nil
}

func (d *InvoiceDiscount) UnmarshalJSON(bytes []byte) error {
	serde := &struct {
		Type InvoiceDiscountType `json:"type"`
	}{}

	if err := json.Unmarshal(bytes, serde); err != nil {
		return fmt.Errorf("failed to JSON deserialize Price type: %w", err)
	}

	switch serde.Type {
	case PercentageDiscountType:
		pct := &InvoiceDiscountPercentage{}
		if err := json.Unmarshal(bytes, pct); err != nil {
			return fmt.Errorf("failed to JSON deserialize percentage discount: %w", err)
		}

		d.t = serde.Type
		d.percentage = pct
	default:
		return fmt.Errorf("unsupported discount type %s", serde.Type)
	}

	return nil
}

func (d *InvoiceDiscount) Type() InvoiceDiscountType {
	return d.t
}

func (d *InvoiceDiscount) Validate() error {
	switch d.t {
	case PercentageDiscountType:
		return d.percentage.Validate()
	default:
		return fmt.Errorf("unsupported discount type %s", d.t)
	}
}

func (d *InvoiceDiscount) DiscountBase() (InvoiceDiscountBase, error) {
	switch d.t {
	case PercentageDiscountType:
		return d.percentage.InvoiceDiscountBase, nil
	default:
		return InvoiceDiscountBase{}, fmt.Errorf("unsupported discount type %s", d.t)
	}
}

func (d *InvoiceDiscount) Equals(other InvoiceDiscount) bool {
	if d.t != other.t {
		return false
	}

	switch d.t {
	case PercentageDiscountType:
		return d.percentage.Equals(*other.percentage)
	default:
		return true
	}
}

func (d *InvoiceDiscount) Clone() InvoiceDiscount {
	clone := InvoiceDiscount{
		t: d.t,
	}

	switch d.t {
	case PercentageDiscountType:
		clone.percentage = d.percentage.Clone()
	}

	return clone
}

type InvoiceDiscounts struct {
	mo.Option[[]InvoiceDiscount]
}

func NewInvoiceDiscounts(v []InvoiceDiscount) InvoiceDiscounts {
	// Normalize empty slice to nil for equality checking
	if len(v) == 0 {
		v = nil
	}

	return InvoiceDiscounts{mo.Some(v)}
}

func (d InvoiceDiscounts) Clone() InvoiceDiscounts {
	if d.IsAbsent() {
		return InvoiceDiscounts{}
	}

	clone := make([]InvoiceDiscount, 0, len(d.OrEmpty()))
	for _, disc := range d.OrEmpty() {
		clone = append(clone, disc.Clone())
	}

	return NewInvoiceDiscounts(clone)
}

func (d *InvoiceDiscounts) Append(discounts ...InvoiceDiscount) {
	d.Option = mo.Some(append(d.OrEmpty(), discounts...))
}

func (d InvoiceDiscounts) Validate() error {
	if d.IsAbsent() {
		return nil
	}

	return errors.Join(lo.Map(d.OrEmpty(), func(discount InvoiceDiscount, idx int) error {
		return ValidationWithFieldPrefix(fmt.Sprintf("%d", idx), discount.Validate())
	})...)
}
