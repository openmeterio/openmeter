package billing

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

// InvoiceCustomer implements the streaming.CustomerUsageAttribution interface
// This is used to query the usage of a customer in a meter query
var _ streaming.Customer = &InvoiceCustomer{}

// NewInvoiceCustomer creates a new InvoiceCustomer from a customer.Customer
func NewInvoiceCustomer(cust customer.Customer) InvoiceCustomer {
	ic := InvoiceCustomer{
		Key:            cust.Key,
		CustomerID:     cust.ID,
		Name:           cust.Name,
		BillingAddress: cust.BillingAddress,
	}

	// If the customer has a usage attribution, we add it to the invoice customer
	// We use the validator but this is not an error, we allow non usage based invoices without usage attribution.
	if err := cust.GetUsageAttribution().Validate(); err == nil {
		ic.UsageAttribution = lo.ToPtr(cust.GetUsageAttribution())
	}

	return ic
}

// InvoiceCustomer represents a customer that is used in an invoice
// We use a specific model as we snapshot the customer at the time of invoice creation,
// and we don't want to modify the customer entity after it has been sent to the customer.
type InvoiceCustomer struct {
	Key              *string                             `json:"key,omitempty"`
	CustomerID       string                              `json:"customerId,omitempty"`
	Name             string                              `json:"name"`
	BillingAddress   *models.Address                     `json:"billingAddress,omitempty"`
	UsageAttribution *streaming.CustomerUsageAttribution `json:"usageAttribution,omitempty"`
}

// GetUsageAttribution returns the customer usage attribution
// implementing the streaming.CustomerUsageAttribution interface
func (c InvoiceCustomer) GetUsageAttribution() streaming.CustomerUsageAttribution {
	subjectKeys := []string{}
	if c.UsageAttribution != nil {
		subjectKeys = c.UsageAttribution.SubjectKeys
	}

	return streaming.NewCustomerUsageAttribution(c.CustomerID, c.Key, subjectKeys)
}

// Validate validates the invoice customer
func (i *InvoiceCustomer) Validate() error {
	if i.CustomerID == "" {
		return fmt.Errorf("customerID is required")
	}

	if i.Key != nil && *i.Key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if i.Name == "" {
		return fmt.Errorf("name is required")
	}

	return nil
}

type CustomerMetadata struct {
	Name string `json:"name"`
}
