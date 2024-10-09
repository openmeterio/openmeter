package httpdriver

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

// newCreateCustomerInput creates a new customer.CreateCustomerInput.
func newCreateCustomerInput(namespace string, apiCustomer api.Customer) customerentity.CreateCustomerInput {
	return customerentity.CreateCustomerInput{
		Namespace: namespace,
		Customer:  newFromAPICustomer(namespace, apiCustomer),
	}
}

// newUpdateCustomerInput creates a new customer.UpdateCustomerInput.
func newUpdateCustomerInput(namespace string, apiCustomer api.Customer) customerentity.UpdateCustomerInput {
	return customerentity.UpdateCustomerInput{
		Namespace: namespace,
		Customer:  newFromAPICustomer(namespace, apiCustomer),
	}
}

// newFromAPICustomer creates a new customer.Customer from an api.Customer.
func newFromAPICustomer(namespace string, apiCustomer api.Customer) customerentity.Customer {
	customerModel := customerentity.Customer{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
		},
		Name:             apiCustomer.Name,
		UsageAttribution: customerentity.CustomerUsageAttribution(apiCustomer.UsageAttribution),
		PrimaryEmail:     apiCustomer.PrimaryEmail,
	}

	if apiCustomer.BillingAddress != nil {
		address := models.Address{
			City:        apiCustomer.BillingAddress.City,
			State:       apiCustomer.BillingAddress.State,
			PostalCode:  apiCustomer.BillingAddress.PostalCode,
			Line1:       apiCustomer.BillingAddress.Line1,
			Line2:       apiCustomer.BillingAddress.Line2,
			PhoneNumber: apiCustomer.BillingAddress.PhoneNumber,
		}

		if apiCustomer.BillingAddress.Country != nil {
			address.Country = lo.ToPtr(models.CountryCode(*apiCustomer.BillingAddress.Country))
		}

		customerModel.BillingAddress = &address
	}

	if apiCustomer.Currency != nil {
		customerModel.Currency = lo.ToPtr(currencyx.Code(*apiCustomer.Currency))
	}

	if apiCustomer.Timezone != nil {
		customerModel.Timezone = lo.ToPtr(timezone.Timezone(*apiCustomer.Timezone))
	}

	return customerModel
}
