package customer

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Customer represents a customer
type Customer struct {
	models.ManagedResource

	UsageAttribution  CustomerUsageAttribution  `json:"usageAttribution"`
	PrimaryEmail      *string                   `json:"primaryEmail"`
	Currency          *models.CurrencyCode      `json:"currency"`
	BillingAddress    *models.Address           `json:"billingAddress"`
	TaxProvider       *models.TaxProvider       `json:"taxProvider"`
	InvoicingProvider *models.InvoicingProvider `json:"invoicingProvider"`
	PaymentProvider   *models.PaymentProvider   `json:"paymentProvider"`
	External          *CustomerExternalMapping  `json:"external"`
}

// CustomerUsageAttribution represents the usage attribution for a customer
type CustomerUsageAttribution struct {
	SubjectKeys []string
}

// CustomerExternalMapping represents the external mapping for a customer
type CustomerExternalMapping struct {
	StripeCustomerID *string `json:"stripeCustomerID"`
}

// ListCustomersInput represents the input for the ListCustomers method
type ListCustomersInput struct {
	models.NamespacedModel
	pagination.Page

	IncludeDisabled bool
}

// CreateCustomerInput represents the input for the CreateCustomer method
type CreateCustomerInput struct {
	models.NamespacedModel
	Customer
}

func (i CreateCustomerInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.Key == "" {
		return ValidationError{
			Err: errors.New("customer key is required"),
		}
	}

	if i.Name == "" {
		return ValidationError{
			Err: errors.New("customer name is required"),
		}
	}

	return nil
}

// DeleteCustomerInput represents the input for the DeleteCustomer method
type DeleteCustomerInput struct {
	models.NamespacedModel
	ID string
}

func (i DeleteCustomerInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("customer id is required"),
		}
	}

	return nil
}

// GetCustomerInput represents the input for the GetCustomer method
type GetCustomerInput struct {
	models.NamespacedModel
	ID string
}

func (i GetCustomerInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("customer id is required"),
		}
	}

	return nil
}

// UpdateCustomerInput represents the input for the UpdateCustomer method
type UpdateCustomerInput struct {
	models.NamespacedModel
	Customer
}

func (i UpdateCustomerInput) Validate(_ context.Context, _ Service) error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if i.ID == "" {
		return ValidationError{
			Err: errors.New("customer id is required"),
		}
	}

	return nil
}
