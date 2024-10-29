package billing

import (
	"fmt"
	"time"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

type CreateCustomerOverrideInput struct {
	Namespace string `json:"namespace"`

	CustomerID string `json:"customerID"`
	ProfileID  string `json:"billingProfile,omitempty"`

	Collection billingentity.CollectionOverrideConfig `json:"collection"`
	Invoicing  billingentity.InvoicingOverrideConfig  `json:"invoicing"`
	Payment    billingentity.PaymentOverrideConfig    `json:"payment"`
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

	Collection billingentity.CollectionOverrideConfig `json:"collection"`
	Invoicing  billingentity.InvoicingOverrideConfig  `json:"invoicing"`
	Payment    billingentity.PaymentOverrideConfig    `json:"payment"`
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

type HasCustomerOverrideReferencingProfileAdapterInput genericNamespaceID

func (i HasCustomerOverrideReferencingProfileAdapterInput) Validate() error {
	return genericNamespaceID(i).Validate()
}

type (
	UpsertCustomerOverrideAdapterInput = customerentity.CustomerID
	LockCustomerForUpdateAdapterInput  = customerentity.CustomerID
)
