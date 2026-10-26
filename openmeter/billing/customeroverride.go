package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// CustomerBillingProfileAssignment selects a profile independently of the namespace default.
// A pin to the current default remains a pin when the default changes.
type CustomerBillingProfileAssignment struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	ProfileID string    `json:"profileID"`
}

// CustomerBillingProfile always contains the effective profile on a successful read.
// Empty and deleted persistence rows do not constitute a pin.
type CustomerBillingProfile struct {
	CustomerID       customer.CustomerID               `json:"customerID"`
	AssignedProfile  *CustomerBillingProfileAssignment `json:"assignedProfile,omitempty"`
	EffectiveProfile Profile                           `json:"effectiveProfile"`
	Expand           CustomerBillingProfileExpand      `json:"expand,omitempty"`
	Customer         *customer.Customer                `json:"customer,omitempty"`
}

type SetCustomerBillingProfileInput struct {
	Customer  customer.CustomerID `json:"customer"`
	ProfileID string              `json:"profileID"`
}

func (i SetCustomerBillingProfileInput) Validate() error {
	var errs []error
	if err := i.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if i.ProfileID == "" {
		errs = append(errs, errors.New("profile ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ClearCustomerBillingProfileInput struct {
	Customer customer.CustomerID
}

func (i ClearCustomerBillingProfileInput) Validate() error {
	return i.Customer.Validate()
}

type GetCustomerBillingProfileInput struct {
	Customer customer.CustomerID          `json:"customerID"`
	Expand   CustomerBillingProfileExpand `json:"expand,omitempty"`
}

func (i GetCustomerBillingProfileInput) Validate() error {
	return i.Customer.Validate()
}

// CustomerBillingProfileAdapterResult contains only persisted selection data.
// Profile is nil when there is no active pin; default resolution belongs to the service.
type CustomerBillingProfileAdapterResult struct {
	CustomerID      customer.CustomerID
	AssignedProfile *CustomerBillingProfileAssignment
	Profile         *Profile
}

type ListCustomerBillingProfilesAdapterResult = pagination.Result[CustomerBillingProfileAdapterResult]

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
			errors.New("app type is required")),
		)
	}

	return errors.Join(errs...)
}

type ListCustomerBillingProfilesInput struct {
	pagination.Page

	// Warning: We only support a single namespace for now as the default profile handling
	// complicates things. If we need multiple namespace support, I would recommend a different
	// endpoint that doesn't take default namespace into account.
	Namespace                     string                       `json:"namespace"`
	BillingProfiles               []string                     `json:"billingProfile,omitempty"`
	CustomersWithoutPinnedProfile bool                         `json:"customersWithoutPinnedProfile,omitempty"`
	Expand                        CustomerBillingProfileExpand `json:"expand,omitempty"`

	IncludeAllCustomers  bool     `json:"includeAllCustomers,omitempty"`
	CustomerIDs          []string `json:"customerID,omitempty"`
	CustomerName         string   `json:"customerName,omitempty"`
	CustomerKey          string   `json:"customerKey,omitempty"`
	CustomerPrimaryEmail string   `json:"customerPrimaryEmail,omitempty"`

	OrderBy CustomerBillingProfileOrderBy
	Order   sortx.Order
}

func (l ListCustomerBillingProfilesInput) Validate() error {
	if l.Namespace == "" {
		return errors.New("namespace is required")
	}

	if l.CustomersWithoutPinnedProfile {
		if len(l.BillingProfiles) > 0 {
			return errors.New("customersWithoutPinnedProfile cannot be used with billingProfiles")
		}
	}

	return nil
}

type CustomerBillingProfileExpand struct {
	// Apps specifies if the effective profile should include the apps
	Apps bool `json:"apps,omitempty"`

	// Customer specifies if the customer should be included in the response
	Customer bool `json:"customer,omitempty"`
}

var CustomerBillingProfileExpandAll = CustomerBillingProfileExpand{
	Apps:     true,
	Customer: true,
}

type CustomerBillingProfileOrderBy string

const (
	CustomerBillingProfileOrderByCustomerID           CustomerBillingProfileOrderBy = "customerId"
	CustomerBillingProfileOrderByCustomerName         CustomerBillingProfileOrderBy = "customerName"
	CustomerBillingProfileOrderByCustomerKey          CustomerBillingProfileOrderBy = "customerKey"
	CustomerBillingProfileOrderByCustomerPrimaryEmail CustomerBillingProfileOrderBy = "customerPrimaryEmail"
	CustomerBillingProfileOrderByCustomerCreatedAt    CustomerBillingProfileOrderBy = "customerCreatedAt"

	DefaultCustomerBillingProfileOrderBy CustomerBillingProfileOrderBy = CustomerBillingProfileOrderByCustomerID
)

var CustomerBillingProfileOrderByValues = []CustomerBillingProfileOrderBy{
	CustomerBillingProfileOrderByCustomerID,
	CustomerBillingProfileOrderByCustomerName,
	CustomerBillingProfileOrderByCustomerKey,
	CustomerBillingProfileOrderByCustomerPrimaryEmail,
	CustomerBillingProfileOrderByCustomerCreatedAt,
}

func (o CustomerBillingProfileOrderBy) Validate() error {
	if !lo.Contains(CustomerBillingProfileOrderByValues, o) {
		return ValidationError{
			Err: fmt.Errorf("invalid order by: %s", o),
		}
	}

	return nil
}

type ListCustomerBillingProfilesResult = pagination.Result[CustomerBillingProfile]

type BulkAssignCustomersToProfileInput struct {
	ProfileID   ProfileID
	CustomerIDs []customer.CustomerID
}

func (b BulkAssignCustomersToProfileInput) Validate() error {
	var errs []error

	if err := b.ProfileID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("profileID: %w", err))
	}

	if len(b.CustomerIDs) == 0 {
		errs = append(errs, errors.New("customerIDs are required"))
	}

	for i, customerID := range b.CustomerIDs {
		if err := customerID.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("customerIDs[%d]: %w", i, err))
		}

		if customerID.Namespace != b.ProfileID.Namespace {
			errs = append(errs, fmt.Errorf("customerIDs[%d] namespace does not match profileID namespace", i))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
