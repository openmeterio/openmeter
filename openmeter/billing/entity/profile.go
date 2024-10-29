package billingentity

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

// AlignmentKind specifies what governs when an invoice is issued
type AlignmentKind string

type Metadata map[string]string

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

type AppReference struct {
	ID   string                `json:"id"`
	Type appentitybase.AppType `json:"type"`
}

func (a AppReference) Validate() error {
	if a.ID == "" && a.Type == "" {
		return errors.New("id or type is required")
	}

	if a.ID != "" && a.Type != "" {
		return errors.New("only one of id or type is allowed")
	}

	return nil
}

// CollectionConfig groups fields related to item collection.
type CollectionConfig struct {
	Alignment AlignmentKind `json:"alignment"`
	Interval  datex.Period  `json:"period,omitempty"`
}

func (c *CollectionConfig) Validate() error {
	if c.Alignment != AlignmentKindSubscription {
		return fmt.Errorf("invalid alignment: %s", c.Alignment)
	}

	if !c.Interval.IsPositive() {
		return fmt.Errorf("item collection period must be greater or equal to 0")
	}

	return nil
}

// InvoiceConfig groups fields related to invoice settings.
type InvoicingConfig struct {
	AutoAdvance bool         `json:"autoAdvance,omitempty"`
	DraftPeriod datex.Period `json:"draftPeriod,omitempty"`
	DueAfter    datex.Period `json:"dueAfter,omitempty"`
}

func (c *InvoicingConfig) Validate() error {
	if c.DraftPeriod.IsNegative() && c.AutoAdvance {
		return fmt.Errorf("draft period must be greater or equal to 0")
	}

	if c.DueAfter.IsNegative() {
		return fmt.Errorf("due after must be greater or equal to 0")
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
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Address models.Address `json:"address"`
	TaxCode *string        `json:"taxCode,omitempty"`
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

type BaseProfile struct {
	ID        string `json:"id"`
	Namespace string `json:"namespace"`

	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	WorkflowConfig WorkflowConfig `json:"workflow"`

	Supplier SupplierContact `json:"supplier"`

	Default  bool     `json:"default"`
	Metadata Metadata `json:"metadata"`

	AppReferences *ProfileAppReferences `json:"appReferences,omitempty"`
}

func (p BaseProfile) Validate() error {
	if p.Namespace == "" {
		return errors.New("namespace is required")
	}

	if p.Name == "" {
		return errors.New("name is required")
	}

	if err := p.WorkflowConfig.Validate(); err != nil {
		return fmt.Errorf("invalid workflow configuration: %w", err)
	}

	if err := p.Supplier.Validate(); err != nil {
		return fmt.Errorf("invalid supplier: %w", err)
	}

	return nil
}

type Profile struct {
	BaseProfile

	// Optionaly expanded fields
	Apps *ProfileApps `json:"-"`
}

type ProfileApps struct {
	Tax       appentity.App `json:"tax"`
	Invoicing appentity.App `json:"invoicing"`
	Payment   appentity.App `json:"payment"`
}

func (p Profile) Validate() error {
	if err := p.BaseProfile.Validate(); err != nil {
		return err
	}

	return nil
}

func (p Profile) Merge(o *CustomerOverride) Profile {
	p.WorkflowConfig.Collection = CollectionConfig{
		Alignment: lo.FromPtrOr(o.Collection.Alignment, p.WorkflowConfig.Collection.Alignment),
		Interval:  lo.FromPtrOr(o.Collection.Interval, p.WorkflowConfig.Collection.Interval),
	}

	p.WorkflowConfig.Invoicing = InvoicingConfig{
		AutoAdvance: lo.FromPtrOr(o.Invoicing.AutoAdvance, p.WorkflowConfig.Invoicing.AutoAdvance),
		DraftPeriod: lo.FromPtrOr(o.Invoicing.DraftPeriod, p.WorkflowConfig.Invoicing.DraftPeriod),
		DueAfter:    lo.FromPtrOr(o.Invoicing.DueAfter, p.WorkflowConfig.Invoicing.DueAfter),
	}

	p.WorkflowConfig.Payment = PaymentConfig{
		CollectionMethod: lo.FromPtrOr(o.Payment.CollectionMethod, p.WorkflowConfig.Payment.CollectionMethod),
	}

	return p
}

type ProfileWithCustomerDetails struct {
	Profile  Profile                 `json:"profile"`
	Customer customerentity.Customer `json:"customer"`
}

func (p ProfileWithCustomerDetails) Validate() error {
	if err := p.Profile.Validate(); err != nil {
		return fmt.Errorf("invalid profile: %w", err)
	}

	return nil
}

type ProfileAppReferences struct {
	Tax       AppReference `json:"tax"`
	Invoicing AppReference `json:"invoicing"`
	Payment   AppReference `json:"payment"`
}

func (i ProfileAppReferences) Validate() error {
	if err := i.Tax.Validate(); err != nil {
		return fmt.Errorf("invalid tax app reference: %w", err)
	}

	if err := i.Invoicing.Validate(); err != nil {
		return fmt.Errorf("invalid invoicing app reference: %w", err)
	}

	if err := i.Payment.Validate(); err != nil {
		return fmt.Errorf("invalid payment app reference: %w", err)
	}

	return nil
}

type InvoiceWorkflow struct {
	AppReferences          ProfileAppReferences `json:"appReferences"`
	Apps                   *ProfileApps         `json:"apps,omitempty"`
	SourceBillingProfileID string               `json:"sourceBillingProfileId,omitempty"`
	Config                 WorkflowConfig       `json:"config"`
}
