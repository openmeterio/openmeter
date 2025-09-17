package customer

import (
	"errors"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ streaming.Customer = &Customer{}

// Customer represents a customer
type Customer struct {
	models.ManagedResource

	Key              *string                  `json:"key,omitempty"`
	UsageAttribution CustomerUsageAttribution `json:"usageAttribution"`
	PrimaryEmail     *string                  `json:"primaryEmail,omitempty"`
	Currency         *currencyx.Code          `json:"currency,omitempty"`
	BillingAddress   *models.Address          `json:"billingAddress,omitempty"`
	Metadata         *models.Metadata         `json:"metadata,omitempty"`
	Annotation       *models.Annotations      `json:"annotations,omitempty"`

	ActiveSubscriptionIDs []string
}

// GetUsageAttribution returns the customer usage attribution
// implementing the streaming.CustomerUsageAttribution interface
func (c Customer) GetUsageAttribution() streaming.CustomerUsageAttribution {
	return streaming.CustomerUsageAttribution{
		ID:          c.ID,
		Key:         c.Key,
		SubjectKeys: c.UsageAttribution.SubjectKeys,
	}
}

// GetID returns the customer id
// This is a convenience method to get the customer id as a CustomerID
// It is used to avoid having to create a CustomerID struct in the codebase
func (c Customer) GetID() CustomerID {
	return CustomerID{
		Namespace: c.Namespace,
		ID:        c.ID,
	}
}

func (c Customer) IsDeleted() bool {
	return c.DeletedAt != nil && c.DeletedAt.Before(clock.Now())
}

// Validate validates the customer
func (c Customer) Validate() error {
	if err := c.ManagedResource.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	if c.Key != nil && *c.Key == "" {
		return models.NewGenericValidationError(errors.New("key cannot be empty"))
	}

	if c.Currency != nil {
		if err := c.Currency.Validate(); err != nil {
			return models.NewGenericValidationError(err)
		}
	}
	return nil
}

type CustomerMutate struct {
	Key              *string                  `json:"key,omitempty"`
	Name             string                   `json:"name"`
	Description      *string                  `json:"description,omitempty"`
	UsageAttribution CustomerUsageAttribution `json:"usageAttribution"`
	PrimaryEmail     *string                  `json:"primaryEmail"`
	Currency         *currencyx.Code          `json:"currency"`
	BillingAddress   *models.Address          `json:"billingAddress"`
	Metadata         *models.Metadata         `json:"metadata"`
	Annotation       *models.Annotations      `json:"annotations,omitempty"`
}

func (c CustomerMutate) Validate() error {
	if c.Key != nil && *c.Key == "" {
		return models.NewGenericValidationError(errors.New("key cannot be empty"))
	}

	if c.Name == "" {
		return models.NewGenericValidationError(errors.New("name is required"))
	}

	if c.Currency != nil {
		if err := c.Currency.Validate(); err != nil {
			return models.NewGenericValidationError(err)
		}
	}
	return nil
}

// CustomerID represents a customer id
type CustomerID models.NamespacedID

func (i CustomerID) Validate() error {
	if i.Namespace == "" {
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	if i.ID == "" {
		return models.NewGenericValidationError(errors.New("customer id is required"))
	}

	return nil
}

// CustomerKey represents a customer key
type CustomerKey struct {
	Namespace string
	Key       string
}

func (i CustomerKey) Validate() error {
	if i.Namespace == "" {
		return models.NewGenericValidationError(errors.New("customer namespace is required"))
	}

	if i.Key == "" {
		return models.NewGenericValidationError(errors.New("customer key is required"))
	}

	return nil
}

// CustomerIDOrKey represents a customer id or key
type CustomerIDOrKey struct {
	Namespace string
	IDOrKey   string
}

func (i CustomerIDOrKey) Validate() error {
	if i.Namespace == "" {
		return models.NewGenericValidationError(errors.New("customer namespace is required"))
	}

	if i.IDOrKey == "" {
		return models.NewGenericValidationError(errors.New("customer idOrKey is required"))
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
		return "", NewErrCustomerSubjectKeyNotSingular(c.SubjectKeys)
	}

	return c.SubjectKeys[0], nil
}

// GetCustomerByUsageAttributionInput represents the input for the GetCustomerByUsageAttribution method
type GetCustomerByUsageAttributionInput struct {
	Namespace  string
	SubjectKey string
}

func (i GetCustomerByUsageAttributionInput) Validate() error {
	if i.Namespace == "" {
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	if i.SubjectKey == "" {
		return models.NewGenericValidationError(errors.New("subject key is required"))
	}

	return nil
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
	CustomerIDs  []string
}

func (i ListCustomersInput) Validate() error {
	if i.Namespace == "" {
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	return nil
}

// ListCustomerUsageAttributionsInput represents the input for the ListCustomerUsageAttributions method

type ListCustomerUsageAttributionsInput struct {
	Namespace string
	pagination.Page

	// Filters
	IncludeDeleted bool
	CustomerIDs    []string
}

func (i ListCustomerUsageAttributionsInput) Validate() error {
	if i.Namespace == "" {
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	return nil
}

// CreateCustomerInput represents the input for the CreateCustomer method
type CreateCustomerInput struct {
	Namespace string
	CustomerMutate
}

func (i CreateCustomerInput) Validate() error {
	if i.Namespace == "" {
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	if err := i.CustomerMutate.Validate(); err != nil {
		return models.NewGenericValidationError(err)
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
		return models.NewGenericValidationError(err)
	}

	if err := i.CustomerMutate.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	return nil
}

// DeleteCustomerInput represents the input for the DeleteCustomer method
type DeleteCustomerInput = CustomerID

// GetCustomerInput represents the input for the GetCustomer method
type GetCustomerInput struct {
	CustomerID      *CustomerID
	CustomerKey     *CustomerKey
	CustomerIDOrKey *CustomerIDOrKey

	// Expand
	Expand []api.CustomerExpand
}

func (i GetCustomerInput) Validate() error {
	var errs []error

	// At least one of the three fields is required
	if i.CustomerID == nil && i.CustomerKey == nil && i.CustomerIDOrKey == nil {
		return models.NewGenericValidationError(errors.New("customer id or key is required"))
	}

	// Only one of the three fields can be provided
	if i.CustomerID != nil && i.CustomerKey != nil {
		return models.NewGenericValidationError(errors.New("customer id and key cannot be provided at the same time"))
	}

	if i.CustomerID != nil && i.CustomerIDOrKey != nil {
		return models.NewGenericValidationError(errors.New("customer id and idOrKey cannot be provided at the same time"))
	}

	if i.CustomerKey != nil && i.CustomerIDOrKey != nil {
		return models.NewGenericValidationError(errors.New("customer key and idOrKey cannot be provided at the same time"))
	}

	// Validate the fields
	if i.CustomerID != nil {
		errs = append(errs, i.CustomerID.Validate())
	}

	if i.CustomerKey != nil {
		errs = append(errs, i.CustomerKey.Validate())
	}

	if i.CustomerIDOrKey != nil {
		errs = append(errs, i.CustomerIDOrKey.Validate())
	}

	return errors.Join(errs...)
}

type GetEntitlementValueInput struct {
	CustomerID CustomerID
	FeatureKey string
}

func (i GetEntitlementValueInput) Validate() error {
	if err := i.CustomerID.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	if i.FeatureKey == "" {
		return models.NewGenericValidationError(errors.New("feature key is required"))
	}

	return nil
}
