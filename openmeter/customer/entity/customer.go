package customerentity

import (
	"errors"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

// Customer represents a customer
type Customer struct {
	models.ManagedResource

	Timezone         *timezone.Timezone       `json:"timezone"`
	UsageAttribution CustomerUsageAttribution `json:"usageAttribution"`
	PrimaryEmail     *string                  `json:"primaryEmail"`
	Currency         *currencyx.Code          `json:"currency"`
	BillingAddress   *models.Address          `json:"billingAddress"`
	Apps             []CustomerApp            `json:"apps"`
}

func (c Customer) Validate() error {
	if c.Name == "" {
		return ValidationError{
			Err: errors.New("name is required"),
		}
	}

	if c.Timezone != nil {
		if err := c.Timezone.Validate(); err != nil {
			return ValidationError{
				Err: err,
			}
		}
	}

	if c.Currency != nil {
		if err := c.Currency.Validate(); err != nil {
			return ValidationError{
				Err: err,
			}
		}
	}
	return nil
}

func (c Customer) GetID() CustomerID {
	return CustomerID{c.Namespace, c.ID}
}

// AsAPICustomer converts a Customer to an API Customer
func (c Customer) AsAPICustomer() (api.Customer, error) {
	apiCustomer := api.Customer{
		Id:               c.ManagedResource.ID,
		Name:             c.Name,
		UsageAttribution: api.CustomerUsageAttribution{SubjectKeys: c.UsageAttribution.SubjectKeys},
		PrimaryEmail:     c.PrimaryEmail,
		Description:      c.Description,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
		DeletedAt:        c.DeletedAt,
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

		apiCustomer.BillingAddress = &address
	}

	if c.Currency != nil {
		apiCustomer.Currency = lo.ToPtr(string(*c.Currency))
	}

	return apiCustomer, nil
}

type CustomerMutate struct {
	Name             string                   `json:"name"`
	Description      *string                  `json:"description,omitempty"`
	Timezone         *timezone.Timezone       `json:"timezone"`
	UsageAttribution CustomerUsageAttribution `json:"usageAttribution"`
	PrimaryEmail     *string                  `json:"primaryEmail"`
	Currency         *currencyx.Code          `json:"currency"`
	BillingAddress   *models.Address          `json:"billingAddress"`
	Apps             []CustomerApp            `json:"apps"`
}

func (c CustomerMutate) Validate() error {
	if c.Name == "" {
		return ValidationError{
			Err: errors.New("name is required"),
		}
	}

	if c.Timezone != nil {
		if err := c.Timezone.Validate(); err != nil {
			return ValidationError{
				Err: err,
			}
		}
	}

	if c.Currency != nil {
		if err := c.Currency.Validate(); err != nil {
			return ValidationError{
				Err: err,
			}
		}
	}
	return nil
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

	// Order
	OrderBy api.CustomerOrderBy
	Order   sortx.Order

	// Filters
	Name         *string
	PrimaryEmail *string
	Subject      *string
}

// CreateCustomerInput represents the input for the CreateCustomer method
type CreateCustomerInput struct {
	Namespace string
	CustomerMutate
}

func (i CreateCustomerInput) Validate() error {
	if i.Namespace == "" {
		return ValidationError{
			Err: errors.New("namespace is required"),
		}
	}

	if err := i.CustomerMutate.Validate(); err != nil {
		return ValidationError{
			Err: err,
		}
	}

	return nil
}

// UpdateCustomerInput represents the input for the UpdateCustomer method
type UpdateCustomerInput struct {
	CustomerID CustomerID
	CustomerMutate
}

func (i UpdateCustomerInput) Validate() error {
	if err := i.CustomerID.Validate(); err != nil {
		return ValidationError{
			Err: err,
		}
	}

	if err := i.CustomerMutate.Validate(); err != nil {
		return ValidationError{
			Err: err,
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
