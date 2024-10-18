package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appshttpdriver "github.com/openmeterio/openmeter/openmeter/app/httpdriver"

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

var DefaultWorkflowConfig = WorkflowConfig{
	Collection: CollectionConfig{
		Alignment: AlignmentKindSubscription,
		Interval:  lo.Must(datex.ISOString("PT2H").Parse()),
	},
	Invoicing: InvoicingConfig{
		AutoAdvance: lo.ToPtr(true),
		DraftPeriod: lo.Must(datex.ISOString("P1D").Parse()),
		DueAfter:    lo.Must(datex.ISOString("P1W").Parse()),
	},
	Payment: PaymentConfig{
		CollectionMethod: CollectionMethodChargeAutomatically,
	},
}

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
			AutoAdvance: c.Invoicing.AutoAdvance,
			DraftPeriod: lo.EmptyableToPtr(c.Invoicing.DraftPeriod.String()),
			DueAfter:    lo.EmptyableToPtr(c.Invoicing.DueAfter.String()),
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
	AutoAdvance *bool        `json:"autoAdvance"`
	DraftPeriod datex.Period `json:"draftPeriod,omitempty"`
	DueAfter    datex.Period `json:"dueAfter"`
}

func (c *InvoicingConfig) Validate() error {
	if c.DraftPeriod.IsNegative() && c.AutoAdvance != nil && *c.AutoAdvance {
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

	WorkflowConfig WorkflowConfig `json:"workflow"`

	Supplier SupplierContact `json:"supplier"`

	Default  bool     `json:"default"`
	Metadata Metadata `json:"metadata"`

	// Optionally expanded fields
	Apps     *ProfileApps             `json:"apps,omitempty"`
	Customer *customerentity.Customer `json:"customer,omitempty"`
}

type AdapterProfile struct {
	Profile

	AppReferences ProfileAppReferences `json:"appReferences"`
}

type ProfileApps struct {
	Tax       appentity.App `json:"tax"`
	Invoicing appentity.App `json:"invoicing"`
	Payment   appentity.App `json:"payment"`
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

// TODO: Make this aprt of the httpdriver instead
func (p Profile) ToAPI() (api.BillingProfile, error) {
	out := api.BillingProfile{
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
	}

	if p.Apps != nil {
		tax, err := appshttpdriver.MapAppToAPI(p.Apps.Tax)
		if err != nil {
			return api.BillingProfile{}, fmt.Errorf("cannot map tax app: %w", err)
		}

		invoicing, err := appshttpdriver.MapAppToAPI(p.Apps.Invoicing)
		if err != nil {
			return api.BillingProfile{}, fmt.Errorf("cannot map invoicing app: %w", err)
		}

		payment, err := appshttpdriver.MapAppToAPI(p.Apps.Payment)
		if err != nil {
			return api.BillingProfile{}, fmt.Errorf("cannot map payment app: %w", err)
		}

		out.Apps = api.BillingProfileApps{
			Tax:       tax,
			Invoicing: invoicing,
			Payment:   payment,
		}
	}

	return out, nil
}

func (p Profile) Merge(o *CustomerOverride) Profile {
	p.WorkflowConfig.Collection = CollectionConfig{
		Alignment: lo.FromPtrOr(o.Collection.Alignment, p.WorkflowConfig.Collection.Alignment),
		Interval:  lo.FromPtrOr(o.Collection.Interval, p.WorkflowConfig.Collection.Interval),
	}

	p.WorkflowConfig.Invoicing = InvoicingConfig{
		AutoAdvance: lo.CoalesceOrEmpty(o.Invoicing.AutoAdvance, p.WorkflowConfig.Invoicing.AutoAdvance),
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

func (i CreateProfileInput) WithDefaults() CreateProfileInput {
	i.WorkflowConfig = WorkflowConfig{
		Collection: CollectionConfig{
			Alignment: lo.CoalesceOrEmpty(
				i.WorkflowConfig.Collection.Alignment,
				DefaultWorkflowConfig.Collection.Alignment),
			Interval: lo.CoalesceOrEmpty(
				i.WorkflowConfig.Collection.Interval,
				DefaultWorkflowConfig.Collection.Interval),
		},
		Invoicing: InvoicingConfig{
			AutoAdvance: lo.CoalesceOrEmpty(
				i.WorkflowConfig.Invoicing.AutoAdvance,
				DefaultWorkflowConfig.Invoicing.AutoAdvance),
			DraftPeriod: lo.CoalesceOrEmpty(
				i.WorkflowConfig.Invoicing.DraftPeriod,
				DefaultWorkflowConfig.Invoicing.DraftPeriod),
			DueAfter: lo.CoalesceOrEmpty(
				i.WorkflowConfig.Invoicing.DueAfter,
				DefaultWorkflowConfig.Invoicing.DueAfter),
		},
		Payment: PaymentConfig{
			CollectionMethod: lo.CoalesceOrEmpty(
				i.WorkflowConfig.Payment.CollectionMethod,
				DefaultWorkflowConfig.Payment.CollectionMethod),
		},
	}

	return i
}

type CreateProfileAppsInput = ProfileAppReferences

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

	if i.Apps != nil {
		return errors.New("apps cannot be updated")
	}

	return Profile(i).Validate()
}
