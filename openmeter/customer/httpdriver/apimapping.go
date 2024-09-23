package httpdriver

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timezone"
	"github.com/samber/lo"
)

// newCreateCustomerInput creates a new customer.CreateCustomerInput.
func newCreateCustomerInput(namespace string, apiCustomer api.Customer) customer.CreateCustomerInput {
	return customer.CreateCustomerInput{
		Namespace: namespace,
		Customer:  newFromAPICustomer(namespace, apiCustomer),
	}
}

// newUpdateCustomerInput creates a new customer.UpdateCustomerInput.
func newUpdateCustomerInput(namespace string, apiCustomer api.Customer) customer.UpdateCustomerInput {
	return customer.UpdateCustomerInput{
		Namespace: namespace,
		Customer:  newFromAPICustomer(namespace, apiCustomer),
	}
}

// newFromAPICustomer creates a new customer.Customer from an api.Customer.
func newFromAPICustomer(namespace string, apiCustomer api.Customer) customer.Customer {
	customerModel := customer.Customer{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{
				Namespace: namespace,
			},
		},
		Name:             apiCustomer.Name,
		UsageAttribution: customer.CustomerUsageAttribution(apiCustomer.UsageAttribution),
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

	if apiCustomer.External != nil {
		external := &customer.CustomerExternalMapping{}

		if apiCustomer.External.StripeCustomerId != nil {
			customerModel.External.StripeCustomerID = apiCustomer.External.StripeCustomerId
		}

		customerModel.External = external
	}

	if apiCustomer.Currency != nil {
		customerModel.Currency = lo.ToPtr(models.CurrencyCode(*apiCustomer.Currency))
	}

	if apiCustomer.Timezone != nil {
		customerModel.Timezone = lo.ToPtr(timezone.Timezone(*apiCustomer.Timezone))
	}

	return customerModel
}
