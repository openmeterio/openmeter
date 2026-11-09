package billing

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/filter"
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
	AutoAdvance                  bool                         `json:"autoAdvance,omitempty"`
	DraftPeriod                  datetime.ISODuration         `json:"draftPeriod,omitempty"`
	DueAfter                     datetime.ISODuration         `json:"dueAfter,omitempty"`
	ProgressiveBilling           bool                         `json:"progressiveBilling,omitempty"`
	SubscriptionEndProrationMode SubscriptionEndProrationMode `json:"subscriptionEndProrationMode,omitempty"`
	// DefaultTaxConfig references the default Tax Code by ID. On an invoice's workflow it's the
	// profile's default frozen at invoice creation; the Stripe code shown for it in v1 responses
	// comes from the Tax Code's current mapping.
	DefaultTaxConfig *productcatalog.TaxCodeConfig `json:"defaultTaxConfig,omitempty"`
}

func (c InvoicingConfig) Clone() InvoicingConfig {
	out := c
	if c.DefaultTaxConfig != nil {
		cloned := c.DefaultTaxConfig.Clone()
		out.DefaultTaxConfig = &cloned
	}

	return out
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

	if c.SubscriptionEndProrationMode == "" {
		return errors.New("subscription end proration mode is required")
	}

	if err := c.SubscriptionEndProrationMode.Validate(); err != nil {
		return fmt.Errorf("invalid subscription end proration mode: %w", err)
	}

	return nil
}

// DefaultTaxCodeID returns the ID of the default Tax Code, or "" when there is none.
func (c InvoicingConfig) DefaultTaxCodeID() string {
	return lo.FromPtr(c.DefaultTaxConfig).TaxCodeID
}

func newDeprecatedDefaultTaxCodeError() error {
	return ValidationError{
		Err: models.NewGenericValidationError(errors.New("setting a tax code (stripe.code / taxCodeId) on a billing profile's defaultTaxConfig is deprecated and can no longer be added or changed; the organization default tax code is used instead. You may still remove it. (behavior is unaffected.)")),
	}
}

// ValidateDefaultTaxCodeUnchanged rejects a default Tax Code ID unless it's the one stored (stored
// is the zero InvoicingConfig on create, so any ID is rejected there). Removing the default and
// changing the behavior are always allowed.
//
// The Tax Code isn't looked up: only the stored ID can pass, and it was valid when it was set.
// Not re-checking it keeps profiles whose default Tax Code was deleted since then editable.
func (c InvoicingConfig) ValidateDefaultTaxCodeUnchanged(stored InvoicingConfig) error {
	id := c.DefaultTaxCodeID()
	if id == "" || id == stored.DefaultTaxCodeID() {
		return nil
	}

	return newDeprecatedDefaultTaxCodeError()
}

type SubscriptionEndProrationMode string

const (
	SubscriptionEndProrationModeBillFullPeriod   SubscriptionEndProrationMode = "bill_full_period"
	SubscriptionEndProrationModeBillActualPeriod SubscriptionEndProrationMode = "bill_actual_period"
)

func (m SubscriptionEndProrationMode) Values() []string {
	return []string{
		string(SubscriptionEndProrationModeBillFullPeriod),
		string(SubscriptionEndProrationModeBillActualPeriod),
	}
}

func (m SubscriptionEndProrationMode) Validate() error {
	if !slices.Contains(m.Values(), string(m)) {
		return fmt.Errorf("invalid subscription end proration mode: %s", m)
	}

	return nil
}

// WithDeprecatedDefaultTaxCodeEnforced returns a ValidationError when the incoming v1/v3 default
// tax config adds or changes a deprecated tax-code field (stripe.code or taxCodeId) relative to
// stored. stored is nil on create. Removal is permitted; behavior is never restricted.
// Call it on the raw API->domain mapped incoming config, BEFORE resolving Stripe codes, with the
// stored default rendered in the same legacy shape.
//
// On a permitted removal it also returns a normalized copy of the incoming config: dropping
// stripe.code while echoing taxCodeId unchanged removes the whole pair (taxCodeId is cleared
// too), otherwise the echoed taxCodeId would keep the Tax Code. The reverse (taxCodeId omitted,
// stripe.code echoed unchanged) is left untouched, because legacy clients that predate taxCodeId
// send exactly that shape on no-op updates.
func WithDeprecatedDefaultTaxCodeEnforced(incoming, stored *productcatalog.TaxConfig) (*productcatalog.TaxConfig, error) {
	if incoming == nil {
		return nil, nil
	}

	i := incoming.Clone()

	taxCodeChanged := i.TaxCodeID != nil &&
		(stored == nil || stored.TaxCodeID == nil || *stored.TaxCodeID != *i.TaxCodeID)

	stripeCodeChanged := i.Stripe != nil && i.Stripe.Code != "" &&
		(stored == nil || stored.Stripe == nil || stored.Stripe.Code != i.Stripe.Code)

	if taxCodeChanged || stripeCodeChanged {
		return nil, newDeprecatedDefaultTaxCodeError()
	}

	stripeCodeRemoved := stored != nil && stored.Stripe != nil && stored.Stripe.Code != "" &&
		(i.Stripe == nil || i.Stripe.Code == "")
	if stripeCodeRemoved {
		i.TaxCodeID = nil
	}

	return &i, nil
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

type ProfileOrderBy string

const (
	ProfileOrderByID        ProfileOrderBy = "id"
	ProfileOrderByName      ProfileOrderBy = "name"
	ProfileOrderByCreatedAt ProfileOrderBy = "created_at"
	ProfileOrderByUpdatedAt ProfileOrderBy = "updated_at"
	ProfileOrderByDefault   ProfileOrderBy = "default"
)

func (o ProfileOrderBy) Values() []ProfileOrderBy {
	return []ProfileOrderBy{
		ProfileOrderByID,
		ProfileOrderByName,
		ProfileOrderByCreatedAt,
		ProfileOrderByUpdatedAt,
		ProfileOrderByDefault,
	}
}

func (o ProfileOrderBy) Validate() error {
	if !slices.Contains(o.Values(), o) {
		return fmt.Errorf("invalid order by value: %s", o)
	}

	return nil
}

type ListProfilesInput struct {
	pagination.Page

	Expand ProfileExpand

	Namespace       string
	IncludeArchived bool
	OrderBy         ProfileOrderBy
	Order           sortx.Order

	ID   *filter.FilterULID
	Name *filter.FilterString
}

func (i ListProfilesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Expand.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("error validating expand: %w", err))
	}

	if i.OrderBy != "" {
		if err := i.OrderBy.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid order by: %w", err))
		}
	}

	if i.ID != nil {
		if err := i.ID.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid id filter: %w", err))
		}
	}

	if i.Name != nil {
		if err := i.Name.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid name filter: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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

	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.AppReferences != nil {
		return errors.New("apps cannot be updated")
	}

	return nil
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
