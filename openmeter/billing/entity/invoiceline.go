package billingentity

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type InvoiceLineType string

const (
	// InvoiceLineTypeManualFee is an item that is manually added to the invoice.
	InvoiceLineTypeManualFee InvoiceLineType = "manual_fee"
	// InvoiceLineTypeFlatFee is an item that is charged at a fixed rate.
	InvoiceLineTypeFlatFee InvoiceLineType = "flat_fee"
	// InvoiceLineTypeUsageBased is an item that is charged based on usage.
	InvoiceLineTypeUsageBased InvoiceLineType = "usage_based"
)

func (InvoiceLineType) Values() []string {
	return []string{
		string(InvoiceLineTypeManualFee),
		string(InvoiceLineTypeFlatFee),
		string(InvoiceLineTypeUsageBased),
	}
}

type InvoiceLineStatus string

const (
	// InvoiceLineStatusValid is a valid invoice line.
	InvoiceLineStatusValid InvoiceLineStatus = "valid"
	// InvoiceLineStatusSplit is a split invoice line (the child lines will have this set as parent).
	InvoiceLineStatusSplit InvoiceLineStatus = "split"
)

func (InvoiceLineStatus) Values() []string {
	return []string{
		string(InvoiceLineStatusValid),
		string(InvoiceLineStatusSplit),
	}
}

type Period struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func (p Period) Validate() error {
	if p.Start.IsZero() {
		return errors.New("start is required")
	}

	if p.End.IsZero() {
		return errors.New("end is required")
	}

	if p.Start.After(p.End) {
		return errors.New("start must be before end")
	}

	return nil
}

// LineBase represents the common fields for an invoice item.
type LineBase struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	Metadata    map[string]string `json:"metadata"`
	Name        string            `json:"name"`
	Type        InvoiceLineType   `json:"type"`
	Description *string           `json:"description,omitempty"`

	InvoiceID string         `json:"invoiceID,omitempty"`
	Currency  currencyx.Code `json:"currency"`

	// Lifecycle
	Period    Period    `json:"period"`
	InvoiceAt time.Time `json:"invoiceAt"`

	// TODO: Add discounts etc

	// Relationships
	ParentLine   *string           `json:"parentLine,omitempty"`
	RelatedLines []string          `json:"relatedLine,omitempty"`
	Status       InvoiceLineStatus `json:"status"`

	TaxOverrides *TaxOverrides `json:"taxOverrides,omitempty"`

	Total alpacadecimal.Decimal `json:"total"`
}

func (i LineBase) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if err := i.Period.Validate(); err != nil {
		return fmt.Errorf("period: %w", err)
	}

	if i.InvoiceAt.IsZero() {
		return errors.New("invoice at is required")
	}

	if i.Name == "" {
		return errors.New("name is required")
	}

	if i.Type == "" {
		return errors.New("type is required")
	}

	if err := i.Currency.Validate(); err != nil {
		return errors.New("currency is required")
	}

	return nil
}

type ManualFeeLine struct {
	Price alpacadecimal.Decimal

	Quantity alpacadecimal.Decimal `json:"quantity"`
}

type Line struct {
	LineBase

	ManualFee *ManualFeeLine `json:"manualFee,omitempty"`
}

func (i Line) Validate() error {
	if err := i.LineBase.Validate(); err != nil {
		return fmt.Errorf("base: %w", err)
	}

	switch i.Type {
	case InvoiceLineTypeManualFee:
		return i.ValidateManualFee()
	default:
		return fmt.Errorf("unsupported type: %s", i.Type)
	}
}

func (i Line) ValidateManualFee() error {
	if i.ManualFee == nil {
		return errors.New("manual fee is required")
	}

	if !i.ManualFee.Price.IsPositive() {
		return errors.New("price should be greater than zero")
	}

	if !i.ManualFee.Quantity.IsPositive() {
		return errors.New("quantity should be positive required")
	}

	// TODO: Validate currency specifics
	return nil
}
