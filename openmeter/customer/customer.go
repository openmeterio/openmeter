package customer

import (
	"errors"
	"fmt"
	"slices"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	Expand  string
	Expands []Expand
)

const (
	ExpandSubscriptions Expand = "subscriptions"
)

func (e Expands) Validate() error {
	for _, expand := range e {
		if expand != ExpandSubscriptions {
			return models.NewGenericValidationError(fmt.Errorf("invalid expand: %s", expand))
		}
	}

	return nil
}

var _ streaming.Customer = &Customer{}

// Customer represents a customer
type Customer struct {
	models.ManagedResource

	Key              *string                   `json:"key,omitempty"`
	UsageAttribution *CustomerUsageAttribution `json:"usageAttribution,omitempty"`
	PrimaryEmail     *string                   `json:"primaryEmail,omitempty"`
	Currency         *currencyx.Code           `json:"currency,omitempty"`
	BillingAddress   *models.Address           `json:"billingAddress,omitempty"`
	Metadata         *models.Metadata          `json:"metadata,omitempty"`
	Annotation       *models.Annotations       `json:"annotations,omitempty"`

	ActiveSubscriptionIDs mo.Option[[]string]
}

// AsCustomerMutate returns a CustomerMutate from the Customer
func (c Customer) AsCustomerMutate() CustomerMutate {
	mut := CustomerMutate{
		Key:            c.Key,
		Name:           c.Name,
		Description:    c.Description,
		PrimaryEmail:   c.PrimaryEmail,
		Currency:       c.Currency,
		BillingAddress: c.BillingAddress,
		Metadata:       c.Metadata,
		Annotation:     c.Annotation,
	}

	if c.UsageAttribution != nil {
		mut.UsageAttribution = &CustomerUsageAttribution{
			SubjectKeys: c.UsageAttribution.SubjectKeys,
		}
	}

	return mut
}

// GetUsageAttribution returns the customer usage attribution
// implementing the streaming.CustomerUsageAttribution interface
func (c Customer) GetUsageAttribution() streaming.CustomerUsageAttribution {
	subjectKeys := []string{}

	if c.UsageAttribution != nil {
		subjectKeys = c.UsageAttribution.SubjectKeys
	}

	return streaming.NewCustomerUsageAttribution(c.ID, c.Key, subjectKeys)
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

	// Either key or usageAttribution.subjectKeys must be provided
	hasKey := c.Key != nil && *c.Key != ""
	hasSubjectKeys := c.UsageAttribution != nil && len(c.UsageAttribution.SubjectKeys) > 0

	if !hasKey && !hasSubjectKeys {
		return models.NewGenericValidationError(errors.New("either key or usageAttribution.subjectKeys must be provided"))
	}

	if c.UsageAttribution != nil {
		if err := c.UsageAttribution.Validate(); err != nil {
			return err
		}
	}

	return nil
}

type CustomerMutate struct {
	Key              *string                   `json:"key,omitempty"`
	Name             string                    `json:"name"`
	Description      *string                   `json:"description,omitempty"`
	UsageAttribution *CustomerUsageAttribution `json:"usageAttribution,omitempty"`
	PrimaryEmail     *string                   `json:"primaryEmail"`
	Currency         *currencyx.Code           `json:"currency"`
	BillingAddress   *models.Address           `json:"billingAddress"`
	Metadata         *models.Metadata          `json:"metadata"`
	Annotation       *models.Annotations       `json:"annotations,omitempty"`
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

	// Either key or usageAttribution.subjectKeys must be provided
	hasKey := c.Key != nil && *c.Key != ""
	hasSubjectKeys := c.UsageAttribution != nil && len(c.UsageAttribution.SubjectKeys) > 0

	if !hasKey && !hasSubjectKeys {
		return models.NewGenericValidationError(errors.New("either key or usageAttribution.subjectKeys must be provided"))
	}

	if c.UsageAttribution != nil {
		if err := c.UsageAttribution.Validate(); err != nil {
			return err
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

var _ models.Validator = (*CustomerUsageAttribution)(nil)

// CustomerUsageAttribution represents the additional fields for a customer usage attribution
// Do not use this struct directly, use the GetUsageAttribution method instead that implements the streaming.CustomerUsageAttribution interface
// The customer usage attribution is more than just the subject keys, it also includes key for example.
type CustomerUsageAttribution struct {
	SubjectKeys []string `json:"subjectKeys"`
}

const MinSubjectKeyLength = 1

func (c CustomerUsageAttribution) Validate() error {
	var errs []error

	for _, subjectKey := range c.SubjectKeys {
		if len(subjectKey) < MinSubjectKeyLength {
			errs = append(errs, models.NewGenericValidationError(
				fmt.Errorf("subject key must be at least %d character: %q", MinSubjectKeyLength, subjectKey),
			))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Deprecated: This functionality is only present for backwards compatibility
func (c CustomerUsageAttribution) GetFirstSubjectKey() (string, error) {
	if len(c.SubjectKeys) == 0 {
		return "", models.NewGenericValidationError(errors.New("no subject keys found"))
	}

	sortedKeys := slices.Clone(c.SubjectKeys)
	slices.Sort(sortedKeys)

	return sortedKeys[0], nil
}

// GetCustomerByUsageAttributionInput represents the input for the GetCustomerByUsageAttribution method
type GetCustomerByUsageAttributionInput struct {
	Namespace string

	// The key of either the customer or one of its subjects
	Key string

	// Expand
	Expands Expands
}

func (i GetCustomerByUsageAttributionInput) Validate() error {
	if i.Namespace == "" {
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	if i.Key == "" {
		return models.NewGenericValidationError(errors.New("subject key is required"))
	}

	if err := i.Expands.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	return nil
}

// ListCustomersInput represents the input for the ListCustomers method
type ListCustomersInput struct {
	Namespace string
	pagination.Page

	IncludeDeleted bool

	// Order
	OrderBy string
	Order   sortx.Order

	// Filters
	Key          *string
	Name         *string
	PrimaryEmail *string
	Subject      *string
	PlanKey      *string
	CustomerIDs  []string

	// Expand
	Expands Expands
}

func (i ListCustomersInput) Validate() error {
	if i.Namespace == "" {
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	if err := i.Expands.Validate(); err != nil {
		return models.NewGenericValidationError(err)
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
	Expands Expands
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

	if err := i.Expands.Validate(); err != nil {
		return models.NewGenericValidationError(err)
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
