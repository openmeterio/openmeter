package customer

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// Customer represents a customer
type Customer struct {
	models.ManagedResource

	Key              *string                  `json:"key,omitempty"`
	UsageAttribution CustomerUsageAttribution `json:"usageAttribution"`
	PrimaryEmail     *string                  `json:"primaryEmail,omitempty"`
	Currency         *currencyx.Code          `json:"currency,omitempty"`
	BillingAddress   *models.Address          `json:"billingAddress,omitempty"`

	CurrentSubscriptionID *string `json:"currentSubscriptionId,omitempty"`
}

func (c Customer) Validate() error {
	if err := c.ManagedResource.Validate(); err != nil {
		return ValidationError{
			Err: err,
		}
	}

	if c.Key != nil && *c.Key == "" {
		return ValidationError{
			Err: errors.New("key cannot be empty"),
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

type CustomerMutate struct {
	Key              *string                  `json:"key,omitempty"`
	Name             string                   `json:"name"`
	Description      *string                  `json:"description,omitempty"`
	UsageAttribution CustomerUsageAttribution `json:"usageAttribution"`
	PrimaryEmail     *string                  `json:"primaryEmail"`
	Currency         *currencyx.Code          `json:"currency"`
	BillingAddress   *models.Address          `json:"billingAddress"`
}

func (c CustomerMutate) Validate() error {
	if c.Key != nil && *c.Key == "" {
		return ValidationError{
			Err: errors.New("key cannot be empty"),
		}
	}

	if c.Name == "" {
		return ValidationError{
			Err: errors.New("name is required"),
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
	SubjectKeys []string `json:"subjectKeys"`
}

// UsageAttribution
func (c CustomerUsageAttribution) GetSubjectKey() (string, error) {
	if len(c.SubjectKeys) != 1 {
		return "", fmt.Errorf("subject mapping is not deterministic, found %d subject keys", len(c.SubjectKeys))
	}

	return c.SubjectKeys[0], nil
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
	Key          *string
	Name         *string
	PrimaryEmail *string
	Subject      *string
	PlanKey      *string
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

type GetEntitlementValueInput struct {
	ID         models.NamespacedID
	FeatureKey string
}

func (i GetEntitlementValueInput) Validate() error {
	if err := i.ID.Validate(); err != nil {
		return ValidationError{
			Err: err,
		}
	}

	if i.FeatureKey == "" {
		return ValidationError{
			Err: errors.New("feature key is required"),
		}
	}

	return nil
}
