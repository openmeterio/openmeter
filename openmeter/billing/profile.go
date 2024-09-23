package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/provider"
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

// TODO: billingconfiguration?!
type Configuration struct {
	// Collection describes the rules for collecting pending line items
	ItemCollection *ItemCollectionConfig `json:"itemCollection,omitempty"`
	// Workflow describes the rules for advancing the billing process
	Workflow *WorkflowConfig `json:"workflow,omitempty"`
	// Granuality describes the rules for line item granuality
	Granuality *GranualityConfig `json:"granuality,omitempty"`
}

func (c *Configuration) Validate() error {
	if c.ItemCollection != nil {
		if err := c.ItemCollection.Validate(); err != nil {
			return fmt.Errorf("failed to validate item collection: %w", err)
		}
	}

	if c.Workflow != nil {
		if err := c.Workflow.Validate(); err != nil {
			return fmt.Errorf("failed to validate workflow: %w", err)
		}
	}

	if c.Granuality != nil {
		if err := c.Granuality.Validate(); err != nil {
			return fmt.Errorf("failed to validate granuality: %w", err)
		}
	}

	return nil
}

type ItemCollectionConfig struct {
	Period time.Duration `json:"period,omitempty"`
}

func (c *ItemCollectionConfig) Validate() error {
	if c.Period < 0 {
		return fmt.Errorf("period must be greater or equal to 0")
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

type WorkflowConfig struct {
	// AutoAdvance specifies if the workflow will automatically advance from draft to issued after DraftPeriod, if not set it
	// will default to the billing provider's default behavior
	AutoAdvance bool `json:"autoAdvance,omitempty"`
	// DraftPeriod specifies how long to wait before automatically advancing from draft to issued
	DraftPeriod time.Duration `json:"draftPeriod,omitempty"`
	// DueAfter specifies how long after the invoice is issued that it is due
	DueAfter time.Duration `json:"dueAfter,omitempty"`
	// CollectionMethod specifies how the invoice should be collected
	CollectionMethod CollectionMethod `json:"collectionMethod,omitempty"`
}

func (c *WorkflowConfig) Validate() error {
	if c.DraftPeriod < 0 && c.AutoAdvance {
		return fmt.Errorf("draft period must be greater or equal to 0")
	}

	if c.DueAfter < 0 {
		return fmt.Errorf("due after must be greater or equal to 0")
	}

	switch c.CollectionMethod {
	case CollectionMethodChargeAutomatically, CollectionMethodSendInvoice:
	default:
		return fmt.Errorf("invalid collection method: %s", c.CollectionMethod)
	}

	return nil
}

type GranualityResolution string

const (
	// GranualityResolutionDay provides line items for metered data per day
	GranualityResolutionDay GranualityResolution = "day"
	// GranualityResolutionPeriod provides one line item per period
	GranualityResolutionPeriod GranualityResolution = "period"
)

func (r GranualityResolution) Values() []string {
	return []string{
		string(GranualityResolutionDay),
		string(GranualityResolutionPeriod),
	}
}

type GranualityConfig struct {
	// Resolution specifies the resolution of the line items
	Resolution GranualityResolution `json:"resolution,omitempty"`
	// PerSubjectDetails specifies if the line items should be split per subject or not
	PerSubjectDetails bool `json:"perSubjectDetails,omitempty"`
}

func (c *GranualityConfig) Validate() error {
	switch c.Resolution {
	case GranualityResolutionDay, GranualityResolutionPeriod:
	default:
		return fmt.Errorf("invalid resolution: %s", c.Resolution)
	}

	return nil
}

type Profile struct {
	ID        string
	Namespace string
	Key       string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	TaxConfiguration       provider.TaxConfiguration
	InvoicingConfiguration provider.InvoicingConfiguration
	PaymentConfiguration   provider.PaymentConfiguration

	Configuration Configuration
	Default       bool
}

func (p Profile) Validate() error {
	// TODO: this is mostly matching the create except of the ID part
	if p.ID == "" {
		return errors.New("id is required")
	}

	if p.Namespace == "" {
		return errors.New("namespace is required")
	}

	if p.Key == "" {
		return errors.New("key is required")
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

	if err := p.Configuration.Validate(); err != nil {
		return fmt.Errorf("invalid workflow configuration: %w", err)
	}
	return nil
}

type CreateProfileInput struct {
	Profile
}

func (i CreateProfileInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Key == "" {
		return errors.New("key is required")
	}

	if err := i.TaxConfiguration.Validate(); err != nil {
		return fmt.Errorf("invalid tax configuration: %w", err)
	}

	if err := i.InvoicingConfiguration.Validate(); err != nil {
		return fmt.Errorf("invalid invoicing configuration: %w", err)
	}

	if err := i.PaymentConfiguration.Validate(); err != nil {
		return fmt.Errorf("invalid payment configuration: %w", err)
	}

	if err := i.Configuration.Validate(); err != nil {
		return fmt.Errorf("invalid workflow configuration: %w", err)
	}

	return nil
}
