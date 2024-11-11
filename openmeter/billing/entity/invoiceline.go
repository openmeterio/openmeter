package billingentity

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	productcatalogmodel "github.com/openmeterio/openmeter/openmeter/productcatalog/model"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type InvoiceLineType string

const (
	// InvoiceLineTypeFee is an item that represents a single charge without meter backing.
	InvoiceLineTypeFee InvoiceLineType = "flat_fee"
	// InvoiceLineTypeUsageBased is an item that is added to the invoice and is usage based.
	InvoiceLineTypeUsageBased InvoiceLineType = "usage_based"
)

func (InvoiceLineType) Values() []string {
	return []string{
		string(InvoiceLineTypeFee),
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

// Period represents a time period, in billing the time period is always interpreted as
// [start, end) (i.e. start is inclusive, end is exclusive).
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

func (p Period) Truncate(resolution time.Duration) Period {
	return Period{
		Start: p.Start.Truncate(resolution),
		End:   p.End.Truncate(resolution),
	}
}

func (p Period) Equal(other Period) bool {
	return p.Start.Equal(other.Start) && p.End.Equal(other.End)
}

func (p Period) IsEmpty() bool {
	return !p.End.After(p.Start)
}

func (p Period) Contains(t time.Time) bool {
	return t.After(p.Start) && t.Before(p.End)
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
	ParentLineID *string           `json:"parentLine,omitempty"`
	ParentLine   *Line             `json:"parent,omitempty"`
	RelatedLines []string          `json:"relatedLine,omitempty"`
	Status       InvoiceLineStatus `json:"status"`

	TaxConfig *TaxConfig `json:"taxOverrides,omitempty"`

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

	if i.InvoiceAt.Before(i.Period.Start) {
		return errors.New("invoice at must be after period start")
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

type FlatFeeLine struct {
	Amount      alpacadecimal.Decimal
	PaymentTerm productcatalogmodel.PaymentTermType

	Quantity alpacadecimal.Decimal `json:"quantity"`
}

type Line struct {
	LineBase

	FlatFee    FlatFeeLine    `json:"flatFee,omitempty"`
	UsageBased UsageBasedLine `json:"usageBased,omitempty"`
}

func (i Line) Validate() error {
	if err := i.LineBase.Validate(); err != nil {
		return fmt.Errorf("base: %w", err)
	}

	if i.InvoiceAt.Before(i.Period.Truncate(DefaultMeterResolution).Start) {
		return errors.New("invoice at must be after period start")
	}

	switch i.Type {
	case InvoiceLineTypeFee:
		return i.ValidateFee()
	case InvoiceLineTypeUsageBased:
		return i.ValidateUsageBased()
	default:
		return fmt.Errorf("unsupported type: %s", i.Type)
	}
}

func (i Line) ValidateFee() error {
	if !i.FlatFee.Amount.IsPositive() {
		return errors.New("price should be greater than zero")
	}

	if !i.FlatFee.Quantity.IsPositive() {
		return errors.New("quantity should be positive required")
	}

	// TODO[OM-947]: Validate currency specifics
	return nil
}

func (i Line) ValidateUsageBased() error {
	if err := i.UsageBased.Validate(); err != nil {
		return fmt.Errorf("usage based price: %w", err)
	}

	if i.InvoiceAt.Before(i.Period.Truncate(DefaultMeterResolution).End) {
		return errors.New("invoice at must be after period end for usage based line")
	}

	return nil
}

type Price = productcatalogmodel.Price

type UsageBasedLine struct {
	Price      Price                  `json:"price"`
	FeatureKey string                 `json:"featureKey"`
	Quantity   *alpacadecimal.Decimal `json:"quantity"`
}

func (i UsageBasedLine) Validate() error {
	if err := i.Price.Validate(); err != nil {
		return fmt.Errorf("price: %w", err)
	}

	if i.FeatureKey == "" {
		return errors.New("featureKey is required")
	}

	return nil
}
