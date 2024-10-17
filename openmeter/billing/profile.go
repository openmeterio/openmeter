package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
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

// Defaults for workflow settings
const (
	// TODO: validate that these are matching the typespec values
	DefaultCollectionAlignment = AlignmentKindSubscription
	DefaultCollectionInterval  = datex.ISOString("PT2H")

	DefaultInvoicingAutoAdvance    = true
	DefaultInvoicingDraftPeriod    = datex.ISOString("P1D")
	DefaultInvoicingDueAfter       = datex.ISOString("P1W")
	DefaultInvoicingItemPerSubject = false
	DefaultInvoicingItemResolution = GranularityResolutionPeriod

	DefaultPaymentCollectionMethod = CollectionMethodChargeAutomatically
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

func (c WorkflowConfig) ToAPI() api.BillingWorkflow {
	return api.BillingWorkflow{
		Id:        c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		DeletedAt: c.DeletedAt,

		// TODO: Timezone

		Collection: &api.BillingWorkflowCollectionSettings{
			Alignment: (*api.BillingWorkflowCollectionAlignment)(lo.EmptyableToPtr(c.Collection.Alignment)),
			Interval:  lo.EmptyableToPtr(c.Collection.Interval.String()),
		},

		Invoicing: &api.BillingWorkflowInvoicingSettings{
			// TODO: we are loosing the default values here?
			AutoAdvance:    lo.EmptyableToPtr(c.Invoicing.AutoAdvance),
			DraftPeriod:    lo.EmptyableToPtr(c.Invoicing.DraftPeriod.String()),
			DueAfter:       lo.EmptyableToPtr(c.Invoicing.DueAfter.String()),
			ItemResolution: (*api.BillingWorkflowItemResolution)(lo.EmptyableToPtr(string(c.Invoicing.ItemResolution))),
			ItemPerSubject: lo.EmptyableToPtr(c.Invoicing.ItemPerSubject),
		},

		Payment: &api.BillingWorkflowPaymentSettings{
			CollectionMethod: (*api.BillingWorkflowCollectionMethod)(lo.EmptyableToPtr(string(c.Payment.CollectionMethod))),
		},
	}
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

type CreateWorkflowConfigInput struct {
	WorkflowConfig
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
	AutoAdvance bool         `json:"autoAdvance"`
	DraftPeriod datex.Period `json:"draftPeriod,omitempty"`
	DueAfter    datex.Period `json:"dueAfter"`

	ItemResolution GranularityResolution `json:"itemResolution"`
	ItemPerSubject bool                  `json:"itemPerSubject"`
}

func (c *InvoicingConfig) Validate() error {
	if c.DraftPeriod.IsNegative() && c.AutoAdvance {
		return fmt.Errorf("draft period must be greater or equal to 0")
	}

	if c.DueAfter.IsNegative() {
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

func (c SupplierContact) ToAPI() api.BillingParty {
	a := c.Address

	return api.BillingParty{
		Name: lo.EmptyableToPtr(c.Name),
		// TODO: taxID
		Addresses: lo.ToPtr([]api.Address{
			{
				Country:     (*string)(a.Country),
				PostalCode:  a.PostalCode,
				State:       a.State,
				City:        a.City,
				Line1:       a.Line1,
				Line2:       a.Line2,
				PhoneNumber: a.PhoneNumber,
			},
		}),
	}
}

type Profile struct {
	ID        string `json:"id"`
	Namespace string `json:"namespace"`

	Name        string  `json:"name"`
	Description *string `json:"description"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`

	// TODO: App references

	WorkflowConfig WorkflowConfig `json:"workflow"`

	Supplier SupplierContact `json:"supplier"`

	Default  bool     `json:"default"`
	Metadata Metadata `json:"metadata"`
}

func (p Profile) Validate() error {
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

func (p Profile) ToAPI() api.BillingProfile {
	return api.BillingProfile{
		Id:        p.ID,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		DeletedAt: p.DeletedAt,

		Description: p.Description,
		Metadata:    (*api.Metadata)(lo.EmptyableToPtr(p.Metadata)),
		Default:     p.Default,

		Name:     p.Name,
		Supplier: p.Supplier.ToAPI(),
		Workflow: p.WorkflowConfig.ToAPI(),

		// TODO: apps should be synced with the new apps sturcture

		Apps: api.BillingProfileApps{ // TODO:
		},
	}
}

func (p Profile) Merge(o *CustomerOverride) Profile {
	p.WorkflowConfig.Collection = CollectionConfig{
		Alignment: lo.FromPtrOr(o.Collection.Alignment, p.WorkflowConfig.Collection.Alignment),
		Interval:  lo.FromPtrOr(o.Collection.Interval, p.WorkflowConfig.Collection.Interval),
	}

	p.WorkflowConfig.Invoicing = InvoicingConfig{
		AutoAdvance:    lo.FromPtrOr(o.Invoicing.AutoAdvance, p.WorkflowConfig.Invoicing.AutoAdvance),
		DraftPeriod:    lo.FromPtrOr(o.Invoicing.DraftPeriod, p.WorkflowConfig.Invoicing.DraftPeriod),
		DueAfter:       lo.FromPtrOr(o.Invoicing.DueAfter, p.WorkflowConfig.Invoicing.DueAfter),
		ItemResolution: lo.FromPtrOr(o.Invoicing.ItemResolution, p.WorkflowConfig.Invoicing.ItemResolution),
		ItemPerSubject: lo.FromPtrOr(o.Invoicing.ItemPerSubject, p.WorkflowConfig.Invoicing.ItemPerSubject),
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

type CreateProfileInput struct {
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	Description *string           `json:"description"`
	Metadata    map[string]string `json:"metadata"`
	Supplier    SupplierContact   `json:"supplier"`
	Default     bool              `json:"default"`

	WorkflowConfig WorkflowConfig         `json:"workflowConfig"`
	Apps           CreateProfileAppsInput `json:"apps"`
}

func (i CreateProfileInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Name == "" {
		return errors.New("name is required")
	}

	if err := i.Supplier.Validate(); err != nil {
		return fmt.Errorf("invalid supplier: %w", err)
	}

	if err := i.WorkflowConfig.Validate(); err != nil {
		return fmt.Errorf("invalid workflow config: %w", err)
	}

	if err := i.Apps.Validate(); err != nil {
		return fmt.Errorf("invalid apps: %w", err)
	}

	return nil
}

// WithDefaults sets the default values for the profile input if not provided,
// this is useful as the object is pretty big and we don't want to set all the fields
func (i CreateProfileInput) WithDefaults() CreateProfileInput {
	// TODO: is this needed?!
	if i.WorkflowConfig.Invoicing.ItemResolution == "" {
		i.WorkflowConfig.Invoicing.ItemResolution = GranularityResolutionPeriod
	}

	if i.WorkflowConfig.Collection.Alignment == "" {
		i.WorkflowConfig.Collection.Alignment = AlignmentKindSubscription
	}

	if i.WorkflowConfig.Payment.CollectionMethod == "" {
		i.WorkflowConfig.Payment.CollectionMethod = CollectionMethodChargeAutomatically
	}

	return i
}

type CreateProfileAppsInput struct {
	Tax       AppReference `json:"tax"`
	Invoicing AppReference `json:"invoicing"`
	Payment   AppReference `json:"payment"`
}

func (i CreateProfileAppsInput) Validate() error {
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
