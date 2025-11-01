package billing

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// AlignmentKind specifies what governs when an invoice is issued
type AlignmentKind string

type Metadata map[string]string

const (
	// AlignmentKindSubscription specifies that the invoice is issued based on the subscription period (
	// e.g. whenever a due line item is added, it will trigger an invoice generation after the collection period)
	AlignmentKindSubscription AlignmentKind = "subscription"

	// AlignmentKindAnchored specifies that the invoice is issued based on the cadence detail.
	// Using this mode results in separate invoices for each invoicable batch of lines.
	AlignmentKindAnchored AlignmentKind = "anchored"

	// THIS IS NOT YET SUPPORTED (this will enable single-invoice mode in effect)
	// AlignmentKindAnchoredBatch specifies that the invoice is issued based on the cadence detail.
	// Using this mode results in all lines added to the gathering invoice in the period
	// until the next recurrence will be invoiced in a single batch at the next recurrence of the anchor.
	AlignmentKindAnchoredBatch AlignmentKind = "anchored_batch"
)

func (k AlignmentKind) Validate() error {
	if !slices.Contains(k.Values(), string(k)) {
		return fmt.Errorf("invalid alignment kind: %s", k)
	}

	return nil
}

func (k AlignmentKind) Values() []string {
	return []string{
		string(AlignmentKindSubscription),
		string(AlignmentKindAnchored),
	}
}

type AnchoredAlignmentDetail struct {
	Interval datetime.ISODuration `json:"interval"`
	Anchor   time.Time            `json:"anchor"`
}

func (a *AnchoredAlignmentDetail) Validate() error {
	if a == nil {
		return nil
	}

	if !a.Interval.IsPositive() {
		return fmt.Errorf("interval must be greater or equal to 0")
	}

	if a.Anchor.IsZero() {
		return fmt.Errorf("anchor must be set")
	}

	return nil
}

// InvoiceConfig groups fields related to invoice settings.
type InvoicingConfig struct {
	AutoAdvance        bool                      `json:"autoAdvance,omitempty"`
	DraftPeriod        datetime.ISODuration      `json:"draftPeriod,omitempty"`
	DueAfter           datetime.ISODuration      `json:"dueAfter,omitempty"`
	ProgressiveBilling bool                      `json:"progressiveBilling,omitempty"`
	DefaultTaxConfig   *productcatalog.TaxConfig `json:"defaultTaxConfig,omitempty"`
}

func (c *InvoicingConfig) Validate() error {
	if c.DraftPeriod.IsNegative() && c.AutoAdvance {
		return fmt.Errorf("draft period must be greater or equal to 0")
	}

	if c.DueAfter.IsNegative() {
		return fmt.Errorf("due after must be greater or equal to 0")
	}

	if c.DefaultTaxConfig != nil {
		if err := c.DefaultTaxConfig.Validate(); err != nil {
			return fmt.Errorf("invalid tax behavior: %w", err)
		}
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
	CollectionMethod CollectionMethod `json:"collectionMethod"`
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

type ProfileID models.NamespacedID

func (p ProfileID) Validate() error {
	return models.NamespacedID(p).Validate()
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

func (p BaseProfile) ProfileID() ProfileID {
	return ProfileID{
		Namespace: p.Namespace,
		ID:        p.ID,
	}
}

type Profile struct {
	BaseProfile

	// Optionaly expanded fields
	Apps *ProfileApps `json:"-"`
}

type AdapterGetProfileResponse struct {
	BaseProfile

	WorkflowConfigID string `json:"workflowConfigId"`
}

func (r *AdapterGetProfileResponse) BaseProfileOrEmpty() *BaseProfile {
	if r == nil {
		return nil
	}

	return &r.BaseProfile
}

type ProfileApps struct {
	Tax       app.App `json:"tax"`
	Invoicing app.App `json:"invoicing"`
	Payment   app.App `json:"payment"`
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
		AnchoredAlignmentDetail: func() *AnchoredAlignmentDetail {
			if o.Collection.Alignment == nil {
				return p.WorkflowConfig.Collection.AnchoredAlignmentDetail
			}

			// This merge logic assumes that Alignment and AnchoredAlignmentDetail are always set together properly.
			// We utilize this so CustomerOverride.AnchoredAlignmentDetail doesn't have to be Option[*AnchoredAlignmentDetail]
			switch *o.Collection.Alignment {
			case AlignmentKindAnchored:
				return o.Collection.AnchoredAlignmentDetail
			case AlignmentKindSubscription:
				return o.Collection.AnchoredAlignmentDetail
			default:
				return nil
			}
		}(),
		Interval: lo.FromPtrOr(o.Collection.Interval, p.WorkflowConfig.Collection.Interval),
	}

	p.WorkflowConfig.Invoicing = InvoicingConfig{
		AutoAdvance:        lo.FromPtrOr(o.Invoicing.AutoAdvance, p.WorkflowConfig.Invoicing.AutoAdvance),
		DraftPeriod:        lo.FromPtrOr(o.Invoicing.DraftPeriod, p.WorkflowConfig.Invoicing.DraftPeriod),
		DueAfter:           lo.FromPtrOr(o.Invoicing.DueAfter, p.WorkflowConfig.Invoicing.DueAfter),
		ProgressiveBilling: lo.FromPtrOr(o.Invoicing.ProgressiveBilling, p.WorkflowConfig.Invoicing.ProgressiveBilling),
		DefaultTaxConfig:   productcatalog.MergeTaxConfigs(o.Invoicing.DefaultTaxConfig, p.WorkflowConfig.Invoicing.DefaultTaxConfig),
	}

	p.WorkflowConfig.Payment = PaymentConfig{
		CollectionMethod: lo.FromPtrOr(o.Payment.CollectionMethod, p.WorkflowConfig.Payment.CollectionMethod),
	}

	return p
}

type ProfileAppReferences struct {
	Tax       app.AppID `json:"tax"`
	Invoicing app.AppID `json:"invoicing"`
	Payment   app.AppID `json:"payment"`
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

type CreateWorkflowConfigInput struct {
	WorkflowConfig
}

type CreateProfileInput struct {
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
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

type CreateProfileAppsInput = ProfileAppReferences

type ListProfilesResult = pagination.Result[Profile]

type ListProfilesInput struct {
	pagination.Page

	Expand ProfileExpand

	Namespace       string
	IncludeArchived bool
	OrderBy         api.BillingProfileOrderBy
	Order           sortx.Order
}

func (i ListProfilesInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if err := i.Expand.Validate(); err != nil {
		return fmt.Errorf("error validating expand: %w", err)
	}

	return nil
}

type ProfileExpand struct {
	Apps bool
}

var ProfileExpandAll = ProfileExpand{
	Apps: true,
}

func (e ProfileExpand) Validate() error {
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

type GetProfileInput struct {
	Profile ProfileID
	Expand  ProfileExpand
}

func (i GetProfileInput) Validate() error {
	if i.Profile.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Profile.ID == "" {
		return errors.New("id is required")
	}

	return nil
}

type DeleteProfileInput = ProfileID

type UpdateProfileInput BaseProfile

func (i UpdateProfileInput) Validate() error {
	if i.ID == "" {
		return errors.New("id is required")
	}

	if i.AppReferences != nil {
		return errors.New("apps cannot be updated")
	}

	return BaseProfile(i).Validate()
}

func (i UpdateProfileInput) ProfileID() ProfileID {
	return BaseProfile(i).ProfileID()
}

type UpdateProfileAdapterInput struct {
	TargetState      BaseProfile
	WorkflowConfigID string
}

func (i UpdateProfileAdapterInput) Validate() error {
	if err := i.TargetState.Validate(); err != nil {
		return fmt.Errorf("error validating target state profile: %w", err)
	}

	if i.TargetState.ID == "" {
		return fmt.Errorf("id is required")
	}

	if i.WorkflowConfigID == "" {
		return fmt.Errorf("workflow config id is required")
	}

	return nil
}

type GetUnpinnedCustomerIDsWithPaidSubscriptionInput struct {
	Namespace string
}

func (i GetUnpinnedCustomerIDsWithPaidSubscriptionInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	return nil
}
