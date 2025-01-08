package billing

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/pkg/models"
)

type InvoiceDiscountType string

const (
	PercentageDiscountType InvoiceDiscountType = "percentage"
	AmountDiscountType     InvoiceDiscountType = "amount"
)

func (d InvoiceDiscountType) Values() []string {
	return []string{string(PercentageDiscountType), string(AmountDiscountType)}
}

type InvoiceDiscountBase struct {
	models.NamespacedModel
	models.ManagedModel

	ID          string              `json:"id"`
	InvoiceID   string              `json:"invoice_id"`
	Description *string             `json:"description"`
	Type        InvoiceDiscountType `json:"type"`
}

func (d *InvoiceDiscountBase) Validate() error {
	// TODO
	return nil
}

type InvoiceDiscountPercentage struct {
	InvoiceDiscountBase

	Percentage alpacadecimal.Decimal `json:"percentage"`
}

func (d *InvoiceDiscountPercentage) Validate() error {
	if d.Percentage.LessThan(alpacadecimal.Zero) || d.Percentage.GreaterThan(alpacadecimal.NewFromInt(100)) {
		return errors.New("discount percentage must be between 0 and 100")
	}

	return d.InvoiceDiscountBase.Validate()
}

type invoiceDiscount interface {
	json.Marshaler
	json.Unmarshaler
	models.Validator

	Type() InvoiceDiscountType
	AsPercentage() (InvoiceDiscountPercentage, error)
	FromPercentage(InvoiceDiscountPercentage)
}

var _ invoiceDiscount = (*InvoiceDiscount)(nil)

type InvoiceDiscount struct {
	t          InvoiceDiscountType
	percentage *InvoiceDiscountPercentage
}

func NewInvoiceDiscountFrom[T InvoiceDiscountPercentage](v T) *InvoiceDiscount {
	discount := &InvoiceDiscount{}

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

	return nil
}
