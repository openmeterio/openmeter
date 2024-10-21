package billingentity

import (
	"errors"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type InvoiceItemType string

const (
	// InvoiceItemTypeStatic is a static item that is not calculated based on usage.
	InvoiceItemTypeStatic InvoiceItemType = "static"
	// InvoiceItemTypeUsage is an item that is calculated based on usage.
	InvoiceItemTypeUsage InvoiceItemType = "usage"
)

func (InvoiceItemType) Values() []string {
	return []string{
		string(InvoiceItemTypeStatic),
		string(InvoiceItemTypeUsage),
	}
}

type InvoiceItem struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	Metadata   map[string]string `json:"metadata"`
	InvoiceID  *string           `json:"invoiceID,omitempty"`
	CustomerID string            `json:"customer"`

	// Lifecycle
	PeriodStart time.Time `json:"periodStart"`
	PeriodEnd   time.Time `json:"periodEnd"`
	InvoiceAt   time.Time `json:"invoiceAt"`

	// Item details
	Name      string                 `json:"name"`
	Type      InvoiceItemType        `json:"type"`
	Quantity  *alpacadecimal.Decimal `json:"quantity"`
	UnitPrice alpacadecimal.Decimal  `json:"unitPrice"`
	Currency  currencyx.Code         `json:"currency"`

	TaxCodeOverride TaxOverrides `json:"taxCodeOverride"`
}

func (i InvoiceItem) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.CustomerID == "" {
		return errors.New("customer id is required")
	}

	if i.PeriodStart.IsZero() {
		return errors.New("period start is required")
	}

	if i.PeriodEnd.IsZero() {
		return errors.New("period end is required")
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

	if i.Type != InvoiceItemTypeStatic {
		// TODO: support usage items
		return errors.New("only static items are supported")
	}

	if i.Type == InvoiceItemTypeStatic && (i.Quantity == nil || i.Quantity.IsZero()) {
		return errors.New("quantity is required for static items")
	}

	if i.UnitPrice.IsZero() {
		return errors.New("unit price is required")
	}

	if i.Currency == "" {
		return errors.New("currency is required")
	}

	return nil
}
