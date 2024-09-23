package customer

import (
	"context"
	"errors"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

// Customer represents a customer
type Customer struct {
	models.ManagedResource

	Name             string                   `json:"name"`
	Timezone         *timezone.Timezone       `json:"timezone"`
	UsageAttribution CustomerUsageAttribution `json:"usageAttribution"`
	PrimaryEmail     *string                  `json:"primaryEmail"`
	Currency         *models.CurrencyCode     `json:"currency"`
	BillingAddress   *models.Address          `json:"billingAddress"`
	External         *CustomerExternalMapping `json:"external"`
}

// AsAPICustomer converts a Customer to an API Customer
func (c Customer) AsAPICustomer() (api.Customer, error) {
	customer := api.Customer{
		Id:               &c.ManagedResource.ID,
		Name:             c.Name,
		UsageAttribution: api.CustomerUsageAttribution{SubjectKeys: c.UsageAttribution.SubjectKeys},
		PrimaryEmail:     c.PrimaryEmail,
	}

	if c.BillingAddress != nil {
		address := api.Address{
			City:        c.BillingAddress.City,
			State:       c.BillingAddress.State,
			PostalCode:  c.BillingAddress.PostalCode,
			Line1:       c.BillingAddress.Line1,
			Line2:       c.BillingAddress.Line2,
			PhoneNumber: c.BillingAddress.PhoneNumber,
		}

		if c.BillingAddress.Country != nil {
			address.Country = lo.ToPtr(string(*c.BillingAddress.Country))
		}

		customer.BillingAddress = &address
	}

	if c.External != nil {
		customer.External = &api.CustomerExternalMapping{
			StripeCustomerId: c.External.StripeCustomerID,
		}
	}

	if c.Currency != nil {
		customer.Currency = lo.ToPtr(string(*c.Currency))
	}

	return customer, nil
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

	IncludeDeleted bool
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
	ID        string
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

// DeleteCustomerInput represents the input for the DeleteCustomer method
type DeleteCustomerInput CustomerID

func (i DeleteCustomerInput) Validate() error {
	return CustomerID(i).Validate()
}

// GetCustomerInput represents the input for the GetCustomer method
type GetCustomerInput CustomerID

func (i GetCustomerInput) Validate() error {
	return CustomerID(i).Validate()
}
