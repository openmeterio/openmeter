package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type CustomerOverride struct {
	Namespace string `json:"namespace"`
	ID        string `json:"id"`

	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	CustomerID string   `json:"customerID"`
	Profile    *Profile `json:"billingProfile,omitempty"`

	Collection CollectionOverrideConfig `json:"collection"`
	Invoicing  InvoicingOverrideConfig  `json:"invoicing"`
	Payment    PaymentOverrideConfig    `json:"payment"`
}

func (c CustomerOverride) Validate() error {
	if c.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if c.ID == "" {
		return fmt.Errorf("id is required")
	}

	if c.CustomerID == "" {
		return fmt.Errorf("customer id is required")
	}

	if c.Profile != nil {
		if err := c.Profile.Validate(); err != nil {
			return fmt.Errorf("invalid profile: %w", err)
		}
	}

	if err := c.Collection.Validate(); err != nil {
		return fmt.Errorf("invalid collection: %w", err)
	}

	if err := c.Invoicing.Validate(); err != nil {
		return fmt.Errorf("invalid invoicing: %w", err)
	}

	if err := c.Payment.Validate(); err != nil {
		return fmt.Errorf("invalid payment: %w", err)
	}

	return nil
}

type CollectionOverrideConfig struct {
	Alignment               *AlignmentKind           `json:"alignment,omitempty"`
	AnchoredAlignmentDetail *AnchoredAlignmentDetail `json:"anchoredAlignmentDetail,omitempty"`
	Interval                *datetime.ISODuration    `json:"interval,omitempty"`
}

func (c *CollectionOverrideConfig) Validate() error {
	if c.Alignment != nil {
		if err := c.Alignment.Validate(); err != nil {
			return fmt.Errorf("invalid alignment: %w", err)
		}
	}

	if c.AnchoredAlignmentDetail != nil {
		if c.Alignment == nil {
			return fmt.Errorf("alignment is required when anchored alignment detail is set")
		}

		switch *c.Alignment {
		case AlignmentKindAnchored:
			if err := c.AnchoredAlignmentDetail.Validate(); err != nil {
				return fmt.Errorf("invalid anchored alignment detail: %w", err)
			}

		case AlignmentKindSubscription:
			return fmt.Errorf("anchored alignment detail is not supported when alignment is subscription")
		}
	}

	if c.Interval != nil && c.Interval.IsNegative() {
		return fmt.Errorf("item collection period must be greater or equal to 0")
	}

	return nil
}

type InvoicingOverrideConfig struct {
	AutoAdvance        *bool                     `json:"autoAdvance,omitempty"`
	DraftPeriod        *datetime.ISODuration     `json:"draftPeriod,omitempty"`
	DueAfter           *datetime.ISODuration     `json:"dueAfter,omitempty"`
	ProgressiveBilling *bool                     `json:"progressiveBilling,omitempty"`
	DefaultTaxConfig   *productcatalog.TaxConfig `json:"defaultTaxConfig,omitempty"`
}

func (c *InvoicingOverrideConfig) Validate() error {
	if c.AutoAdvance != nil && *c.AutoAdvance {
		return fmt.Errorf("auto advance is not supported")
	}

	if c.DueAfter != nil && c.DueAfter.IsNegative() {
		return fmt.Errorf("due after must be greater or equal to 0")
	}

	if c.DraftPeriod != nil && c.DraftPeriod.IsNegative() {
		return fmt.Errorf("draft period must be greater or equal to 0")
	}

	if c.DefaultTaxConfig != nil {
		if err := c.DefaultTaxConfig.Validate(); err != nil {
			return fmt.Errorf("invalid default tax config: %w", err)
		}
	}

	return nil
}

type PaymentOverrideConfig struct {
	CollectionMethod *CollectionMethod
}

func (c *PaymentOverrideConfig) Validate() error {
	if c.CollectionMethod != nil {
		switch *c.CollectionMethod {
		case CollectionMethodChargeAutomatically, CollectionMethodSendInvoice:
		default:
			return fmt.Errorf("invalid collection method: %s", *c.CollectionMethod)
		}
	}

	return nil
}

type UpsertCustomerOverrideInput struct {
	Namespace  string `json:"namespace"`
	CustomerID string `json:"customerID"`

	ProfileID string `json:"billingProfileID"`

	Collection CollectionOverrideConfig `json:"collection"`
	Invoicing  InvoicingOverrideConfig  `json:"invoicing"`
	Payment    PaymentOverrideConfig    `json:"payment"`
}

func (u UpsertCustomerOverrideInput) Validate() error {
	if u.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if u.CustomerID == "" {
		return fmt.Errorf("customer id is required")
	}

	if err := u.Collection.Validate(); err != nil {
		return fmt.Errorf("invalid collection: %w", err)
	}

	if err := u.Invoicing.Validate(); err != nil {
		return fmt.Errorf("invalid invoicing: %w", err)
	}

	if err := u.Payment.Validate(); err != nil {
		return fmt.Errorf("invalid payment: %w", err)
	}

	return nil
}

type GetCustomerOverrideInput struct {
	Customer customer.CustomerID    `json:"customerID"`
	Expand   CustomerOverrideExpand `json:"expand,omitempty"`
}

func (g GetCustomerOverrideInput) Validate() error {
	return g.Customer.Validate()
}

// GetCustomerAppInput is used to get a customer app from a customer override
type GetCustomerAppInput struct {
	CustomerID customer.CustomerID
	AppType    app.AppType
}

// Validate validates the input
func (g GetCustomerAppInput) Validate() error {
	var errs []error

	if err := g.CustomerID.Validate(); err != nil {
		errs = append(errs, models.NewGenericPreConditionFailedError(err))
	}

	if g.AppType == "" {
		errs = append(errs, models.NewGenericPreConditionFailedError(
			fmt.Errorf("app type is required")),
		)
	}

	return errors.Join(errs...)
}

type DeleteCustomerOverrideInput struct {
	Customer customer.CustomerID
}

func (d DeleteCustomerOverrideInput) Validate() error {
	return d.Customer.Validate()
}

type GetProfileWithCustomerOverrideInput struct {
	Customer customer.CustomerID
}

func (g GetProfileWithCustomerOverrideInput) Validate() error {
	return g.Customer.Validate()
}

type GetCustomerOverrideAdapterInput struct {
	Customer customer.CustomerID

	IncludeDeleted bool
}

func (i GetCustomerOverrideAdapterInput) Validate() error {
	if err := i.Customer.Validate(); err != nil {
		return fmt.Errorf("error validating customer: %w", err)
	}

	return nil
}

type (
	UpdateCustomerOverrideAdapterInput = UpsertCustomerOverrideInput
	CreateCustomerOverrideAdapterInput = UpdateCustomerOverrideAdapterInput
)

type HasCustomerOverrideReferencingProfileAdapterInput = ProfileID

type CustomerOverrideWithDetails struct {
	CustomerOverride *CustomerOverride `json:",inline"`
	MergedProfile    Profile           `json:"mergedProfile,omitempty"`

	// Expanded fields
	Expand   CustomerOverrideExpand `json:"expand,omitempty"`
	Customer *customer.Customer     `json:"customer,omitempty"`
}

type CustomerOverrideWithAdapterProfile struct {
	CustomerOverride `json:",inline"`

	DefaultProfile *AdapterGetProfileResponse `json:"billingProfile,omitempty"`
}

type ListCustomerOverridesInput struct {
	pagination.Page

	// Warning: We only support a single namespace for now as the default profile handling
	// complicates things. If we need multiple namespace support, I would recommend a different
	// endpoint that doesn't take default namespace into account.
	Namespace                     string                 `json:"namespace"`
	BillingProfiles               []string               `json:"billingProfile,omitempty"`
	CustomersWithoutPinnedProfile bool                   `json:"customersWithoutPinnedProfile,omitempty"`
	Expand                        CustomerOverrideExpand `json:"expand,omitempty"`

	IncludeAllCustomers  bool     `json:"includeAllCustomers,omitempty"`
	CustomerIDs          []string `json:"customerID,omitempty"`
	CustomerName         string   `json:"customerName,omitempty"`
	CustomerKey          string   `json:"customerKey,omitempty"`
	CustomerPrimaryEmail string   `json:"customerPrimaryEmail,omitempty"`

	OrderBy CustomerOverrideOrderBy
	Order   sortx.Order
}

func (l ListCustomerOverridesInput) Validate() error {
	if l.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if l.CustomersWithoutPinnedProfile {
		if len(l.BillingProfiles) > 0 {
			return fmt.Errorf("customersWithoutPinnedProfile cannot be used with billingProfiles")
		}
	}

	return nil
}

type CustomerOverrideExpand struct {
	// Apps specifies if the merged profile should include the apps
	Apps bool `json:"apps,omitempty"`

	// Customer specifies if the customer should be included in the response
	Customer bool `json:"customer,omitempty"`
}

var CustomerOverrideExpandAll = CustomerOverrideExpand{
	Apps:     true,
	Customer: true,
}

type CustomerOverrideOrderBy string

const (
	CustomerOverrideOrderByCustomerID           CustomerOverrideOrderBy = "customerId"
	CustomerOverrideOrderByCustomerName         CustomerOverrideOrderBy = "customerName"
	CustomerOverrideOrderByCustomerKey          CustomerOverrideOrderBy = "customerKey"
	CustomerOverrideOrderByCustomerPrimaryEmail CustomerOverrideOrderBy = "customerPrimaryEmail"
	CustomerOverrideOrderByCustomerCreatedAt    CustomerOverrideOrderBy = "customerCreatedAt"

	DefaultCustomerOverrideOrderBy CustomerOverrideOrderBy = CustomerOverrideOrderByCustomerID
)

var CustomerOverrideOrderByValues = []CustomerOverrideOrderBy{
	CustomerOverrideOrderByCustomerID,
	CustomerOverrideOrderByCustomerName,
	CustomerOverrideOrderByCustomerKey,
	CustomerOverrideOrderByCustomerPrimaryEmail,
	CustomerOverrideOrderByCustomerCreatedAt,
}

func (o CustomerOverrideOrderBy) Validate() error {
	if !lo.Contains(CustomerOverrideOrderByValues, o) {
		return ValidationError{
			Err: fmt.Errorf("invalid order by: %s", o),
		}
	}

	return nil
}

type ListCustomerOverridesResult = pagination.Result[CustomerOverrideWithDetails]

type CustomerOverrideWithCustomerID struct {
	*CustomerOverride `json:",inline"`

	CustomerID customer.CustomerID `json:"customerID,omitempty"`
}

type ListCustomerOverridesAdapterResult = pagination.Result[CustomerOverrideWithCustomerID]

type BulkAssignCustomersToProfileInput struct {
	ProfileID   ProfileID
	CustomerIDs []customer.CustomerID
}

func (b BulkAssignCustomersToProfileInput) Validate() error {
	if err := b.ProfileID.Validate(); err != nil {
		return fmt.Errorf("invalid billing profile: %w", err)
	}

	if len(b.CustomerIDs) == 0 {
		return errors.New("customer ids are required")
	}

	for i, customerID := range b.CustomerIDs {
		if err := customerID.Validate(); err != nil {
			return fmt.Errorf("invalid customer id[%d]: %w", i, err)
		}
	}

	return nil
}
