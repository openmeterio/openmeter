package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/provider"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

// AlignmentKind specifies what governs when an invoice is issued
type AlignmentKind string

const (
	// AlignmentKindSubscription specifies that the invoice is issued based on the subscription period (
	// e.g. whenever a due line item is added, it will trigger an invoice generation after the collection period)
	AlignmentKindSubscription AlignmentKind = "subscription"
)

func (k AlignmentKind) Values() []string {
	return []string{
		string(AlignmentKindSubscription),
	}
}

type WorkflowConfig struct {
	ID string `json:"id"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`

	Timezone *timezone.Timezone `json:"timezone,omitempty"`

	Collection CollectionConfig `json:"collection"`
	Invoicing  InvoicingConfig  `json:"invoicing"`
	Payment    PaymentConfig    `json:"payment"`
}

func (c WorkflowConfig) Validate() error {
	if err := c.Collection.Validate(); err != nil {
		return fmt.Errorf("invalid collection config: %w", err)
	}

	if err := c.Invoicing.Validate(); err != nil {
		return fmt.Errorf("invalid invoice config: %w", err)
	}

	if err := c.Payment.Validate(); err != nil {
		return fmt.Errorf("invalid payment config: %w", err)
	}

	return nil
}

// CollectionConfig groups fields related to item collection.
type CollectionConfig struct {
	Alignment            AlignmentKind `json:"alignment"`
	ItemCollectionPeriod time.Duration `json:"itemCollectionPeriod,omitempty"`
}

func (c *CollectionConfig) Validate() error {
	if c.Alignment != AlignmentKindSubscription {
		return fmt.Errorf("invalid alignment: %s", c.Alignment)
	}

	if c.ItemCollectionPeriod < 0 {
		return fmt.Errorf("item collection period must be greater or equal to 0")
	}

	return nil
}

// InvoiceConfig groups fields related to invoice settings.
type InvoicingConfig struct {
	AutoAdvance bool          `json:"autoAdvance"`
	DraftPeriod time.Duration `json:"draftPeriod,omitempty"`
	DueAfter    time.Duration `json:"dueAfter"`

	ItemResolution GranularityResolution `json:"itemResolution"`
	ItemPerSubject bool                  `json:"itemPerSubject"`
}

func (c *InvoicingConfig) Validate() error {
	if c.DraftPeriod < 0 && c.AutoAdvance {
		return fmt.Errorf("draft period must be greater or equal to 0")
	}

	if c.DueAfter < 0 {
		return fmt.Errorf("due after must be greater or equal to 0")
	}

	switch c.ItemResolution {
	case GranularityResolutionDay, GranularityResolutionPeriod:
	default:
		return fmt.Errorf("invalid line item resolution: %s", c.ItemResolution)
	}

	return nil
}

type GranularityResolution string

const (
	// GranularityResolutionDay provides line items for metered data per day
	GranularityResolutionDay GranularityResolution = "day"
	// GranularityResolutionPeriod provides one line item per period
	GranularityResolutionPeriod GranularityResolution = "period"
)

func (r GranularityResolution) Values() []string {
	return []string{
		string(GranularityResolutionDay),
		string(GranularityResolutionPeriod),
	}
}

type PaymentConfig struct {
	CollectionMethod CollectionMethod
}

func (c *PaymentConfig) Validate() error {
	switch c.CollectionMethod {
	case CollectionMethodChargeAutomatically, CollectionMethodSendInvoice:
	default:
		return fmt.Errorf("invalid collection method: %s", c.CollectionMethod)
	}

	return nil
}

type CollectionMethod string

const (
	// CollectionMethodChargeAutomatically charges the customer automatically based on previously saved card data
	CollectionMethodChargeAutomatically CollectionMethod = "charge_automatically"
	// CollectionMethodSendInvoice sends an invoice to the customer along with the payment instructions/links
	CollectionMethodSendInvoice CollectionMethod = "send_invoice"
)

func (c CollectionMethod) Values() []string {
	return []string{
		string(CollectionMethodChargeAutomatically),
		string(CollectionMethodSendInvoice),
	}
}

type SupplierContact struct {
	Name    string         `json:"name"`
	Address models.Address `json:"address"`
}

// Validate checks if the supplier contact is valid for invoice generation (e.g. Country is required)
func (c SupplierContact) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}

	if c.Address.Country == nil {
		return errors.New("country is required")
	}

	return nil
}

type Profile struct {
	ID        string `json:"id"`
	Namespace string `json:"namespace"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`

	TaxConfiguration       provider.TaxConfiguration       `json:"tax"`
	InvoicingConfiguration provider.InvoicingConfiguration `json:"invoicing"`
	PaymentConfiguration   provider.PaymentConfiguration   `json:"payment"`

	WorkflowConfig WorkflowConfig `json:"workflow"`

	Supplier SupplierContact `json:"supplier"`

	Default bool `json:"default"`
}

func (p Profile) Validate() error {
	if p.Namespace == "" {
		return errors.New("namespace is required")
	}

	if err := p.TaxConfiguration.Validate(); err != nil {
		return fmt.Errorf("invalid tax configuration: %w", err)
	}

	if err := p.InvoicingConfiguration.Validate(); err != nil {
		return fmt.Errorf("invalid invoicing configuration: %w", err)
	}

	if err := p.PaymentConfiguration.Validate(); err != nil {
		return fmt.Errorf("invalid payment configuration: %w", err)
	}

	if err := p.WorkflowConfig.Validate(); err != nil {
		return fmt.Errorf("invalid workflow configuration: %w", err)
	}

	if err := p.Supplier.Validate(); err != nil {
		return fmt.Errorf("invalid supplier: %w", err)
	}

	return nil
}

type CreateProfileInput Profile

func (i CreateProfileInput) Validate() error {
	return Profile(i).Validate()
}

// WithDefaults sets the default values for the profile input if not provided,
// this is useful as the object is pretty big and we don't want to set all the fields
func (i CreateProfileInput) WithDefaults() CreateProfileInput {
	if i.WorkflowConfig.Invoicing.ItemResolution == "" {
		i.WorkflowConfig.Invoicing.ItemResolution = GranularityResolutionPeriod
	}

	if i.WorkflowConfig.Collection.Alignment == "" {
		i.WorkflowConfig.Collection.Alignment = AlignmentKindSubscription
	}

	if i.WorkflowConfig.Payment.CollectionMethod == "" {
		i.WorkflowConfig.Payment.CollectionMethod = CollectionMethodChargeAutomatically
	}

	if i.WorkflowConfig.Invoicing.DueAfter == 0 {
		i.WorkflowConfig.Invoicing.DueAfter = 30 * 24 * time.Hour
	}

	if i.WorkflowConfig.Invoicing.DraftPeriod == 0 {
		i.WorkflowConfig.Invoicing.DraftPeriod = 24 * time.Hour
	}

	return i
}

type GetDefaultProfileInput struct {
	Namespace string
}

func (i GetDefaultProfileInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	return nil
}

type genericNamespaceID struct {
	Namespace string
	ID        string
}

func (i genericNamespaceID) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.ID == "" {
		return errors.New("id is required")
	}

	return nil
}

type GetProfileInput genericNamespaceID

func (i GetProfileInput) Validate() error {
	return genericNamespaceID(i).Validate()
}

type DeleteProfileInput genericNamespaceID

func (i DeleteProfileInput) Validate() error {
	return genericNamespaceID(i).Validate()
}

type UpdateProfileInput Profile

func (i UpdateProfileInput) Validate() error {
	if i.ID == "" {
		return errors.New("id is required")
	}

	return Profile(i).Validate()
}
