package customer

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing/provider"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Customer represents a customer
type Customer struct {
	models.ManagedResource

	Name              string                      `json:"name"`
	UsageAttribution  CustomerUsageAttribution    `json:"usageAttribution"`
	PrimaryEmail      *string                     `json:"primaryEmail"`
	Currency          *currencyx.Code             `json:"currency"`
	BillingAddress    *models.Address             `json:"billingAddress"`
	TaxProvider       *provider.TaxProvider       `json:"taxProvider"`
	InvoicingProvider *provider.InvoicingProvider `json:"invoicingProvider"`
	PaymentProvider   *provider.PaymentProvider   `json:"paymentProvider"`
	External          *CustomerExternalMapping    `json:"external"`
}

type CustomerID models.NamespacedID

func (i CustomerID) Validate() error {
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
	Namespace string
	pagination.Page

	IncludeDisabled bool
}

// CreateCustomerInput represents the input for the CreateCustomer method
type CreateCustomerInput struct {
	Namespace string
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

// UpdateCustomerInput represents the input for the UpdateCustomer method
type UpdateCustomerInput struct {
	Namespace string
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
