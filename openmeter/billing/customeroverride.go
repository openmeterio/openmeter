package billing

import (
	"fmt"
	"time"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datex"
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
	Alignment *AlignmentKind `json:"alignment,omitempty"`
	Interval  *datex.Period  `json:"interval,omitempty"`
}

func (c *CollectionOverrideConfig) Validate() error {
	if c.Alignment != nil && *c.Alignment != AlignmentKindSubscription {
		return fmt.Errorf("invalid alignment: %s", *c.Alignment)
	}

	if c.Interval != nil && c.Interval.IsNegative() {
		return fmt.Errorf("item collection period must be greater or equal to 0")
	}

	return nil
}

type InvoicingOverrideConfig struct {
	AutoAdvance        *bool                     `json:"autoAdvance,omitempty"`
	DraftPeriod        *datex.Period             `json:"draftPeriod,omitempty"`
	DueAfter           *datex.Period             `json:"dueAfter,omitempty"`
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

type CreateCustomerOverrideInput struct {
	Namespace string `json:"namespace"`

	CustomerID string `json:"customerID"`
	ProfileID  string `json:"billingProfile,omitempty"`

	Collection CollectionOverrideConfig `json:"collection"`
	Invoicing  InvoicingOverrideConfig  `json:"invoicing"`
	Payment    PaymentOverrideConfig    `json:"payment"`
}

func (c CreateCustomerOverrideInput) Validate() error {
	if c.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if c.CustomerID == "" {
		return fmt.Errorf("customer id is required")
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

type UpdateCustomerOverrideInput struct {
	Namespace  string `json:"namespace"`
	CustomerID string `json:"customerID"`

	UpdatedAt time.Time `json:"updatedAt"`

	ProfileID string `json:"billingProfileID"`

	Collection CollectionOverrideConfig `json:"collection"`
	Invoicing  InvoicingOverrideConfig  `json:"invoicing"`
	Payment    PaymentOverrideConfig    `json:"payment"`
}

func (u UpdateCustomerOverrideInput) Validate() error {
	if u.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if u.CustomerID == "" {
		return fmt.Errorf("customer id is required")
	}

	if u.UpdatedAt.IsZero() {
		return fmt.Errorf("updated at is required")
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

type namespacedCustomerID struct {
	Namespace  string `json:"namespace"`
	CustomerID string `json:"customerID"`
}

func (g namespacedCustomerID) Validate() error {
	if g.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if g.CustomerID == "" {
		return fmt.Errorf("customer id is required")
	}

	return nil
}

type GetCustomerOverrideInput namespacedCustomerID

func (g GetCustomerOverrideInput) Validate() error {
	return namespacedCustomerID(g).Validate()
}

type DeleteCustomerOverrideInput namespacedCustomerID

func (d DeleteCustomerOverrideInput) Validate() error {
	return namespacedCustomerID(d).Validate()
}

type GetProfileWithCustomerOverrideInput namespacedCustomerID

func (g GetProfileWithCustomerOverrideInput) Validate() error {
	return namespacedCustomerID(g).Validate()
}

type GetCustomerOverrideAdapterInput struct {
	Customer customerentity.CustomerID

	IncludeDeleted bool
}

func (i GetCustomerOverrideAdapterInput) Validate() error {
	if err := i.Customer.Validate(); err != nil {
		return fmt.Errorf("error validating customer: %w", err)
	}

	return nil
}

type UpdateCustomerOverrideAdapterInput struct {
	UpdateCustomerOverrideInput

	ResetDeletedAt bool
}

func (i UpdateCustomerOverrideAdapterInput) Validate() error {
	if err := i.UpdateCustomerOverrideInput.Validate(); err != nil {
		return fmt.Errorf("error validating update customer override input: %w", err)
	}

	return nil
}

type HasCustomerOverrideReferencingProfileAdapterInput = ProfileID

type (
	UpsertCustomerOverrideAdapterInput = customerentity.CustomerID
	LockCustomerForUpdateAdapterInput  = customerentity.CustomerID
)
