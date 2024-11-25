package httpdriver

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

func MapCustomerCreate(body api.CustomerCreate) customerentity.CustomerMutate {
	return customerentity.CustomerMutate{
		Name:             body.Name,
		Description:      body.Description,
		UsageAttribution: customerentity.CustomerUsageAttribution(body.UsageAttribution),
		PrimaryEmail:     body.PrimaryEmail,
		BillingAddress:   mapAddress(body.BillingAddress),
		Currency:         mapCurrency(body.Currency),
		Timezone:         mapTimezone(body.Timezone),
	}
}

func mapCurrency(apiCurrency *string) *currencyx.Code {
	if apiCurrency == nil {
		return nil
	}

	return lo.ToPtr(currencyx.Code(*apiCurrency))
}

func mapTimezone(apiTimezone *string) *timezone.Timezone {
	if apiTimezone == nil {
		return nil
	}

	return lo.ToPtr(timezone.Timezone(*apiTimezone))
}

func mapAddress(apiAddress *api.Address) *models.Address {
	if apiAddress == nil {
		return nil
	}

	address := models.Address{
		City:        apiAddress.City,
		State:       apiAddress.State,
		PostalCode:  apiAddress.PostalCode,
		Line1:       apiAddress.Line1,
		Line2:       apiAddress.Line2,
		PhoneNumber: apiAddress.PhoneNumber,
	}

	if apiAddress.Country != nil {
		address.Country = lo.ToPtr(models.CountryCode(*apiAddress.Country))
	}

	return &address
}

func mapApp(namespace string, apiApp api.CustomerApp) (customerentity.CustomerApp, error) {
	var customerApp customerentity.CustomerApp

	// Get app type
	appType, err := apiApp.Discriminator()
	if err != nil {
		return customerApp, fmt.Errorf("error getting app type: %w", err)
	}

	switch appType {
	// Sandbox app
	case string(appentitybase.AppTypeSandbox):
		apiSandboxApp, err := apiApp.AsSandboxCustomerApp()
		if err != nil {
			return customerApp, fmt.Errorf("error converting to sandbox app: %w", err)
		}

		customerApp = customerentity.CustomerApp{
			Type: appentitybase.AppTypeSandbox,
			Data: apiSandboxApp.Data,
		}

		if apiSandboxApp.Id != nil {
			customerApp.AppID = &appentitybase.AppID{
				Namespace: namespace,
				ID:        *apiSandboxApp.Id,
			}
		}

	// Stripe app
	case string(appentitybase.AppTypeStripe):
		apiStripeApp, err := apiApp.AsStripeCustomerApp()
		if err != nil {
			return customerApp, fmt.Errorf("error converting to stripe app: %w", err)
		}

		customerApp = customerentity.CustomerApp{
			Type: appentitybase.AppTypeStripe,
			Data: appstripeentity.CustomerAppData{
				StripeCustomerID:             apiStripeApp.Data.StripeCustomerId,
				StripeDefaultPaymentMethodID: apiStripeApp.Data.StripeDefaultPaymentMethodId,
			},
		}

		if apiStripeApp.Id != nil {
			customerApp.AppID = &appentitybase.AppID{
				Namespace: namespace,
				ID:        *apiStripeApp.Id,
			}
		}

	// Unknown app
	default:
		return customerApp, fmt.Errorf("unsupported app type: %s", appType)
	}

	return customerApp, nil
}

// customerToAPI converts a Customer to an API Customer
func customerToAPI(c customerentity.Customer) (api.Customer, error) {
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

	if c.Apps != nil {
		var apiCustomerApps []api.CustomerApp
		for _, customerApp := range c.Apps {
			apiCustomerApp, err := customerAppToAPI(customerApp)
			if err != nil {
				return apiCustomer, fmt.Errorf("error converting customer app to api: %w", err)
			}

			apiCustomerApps = append(apiCustomerApps, apiCustomerApp)
		}

		apiCustomer.Apps = &apiCustomerApps
	}

	return apiCustomer, nil
}

// customerAppToAPI converts a CustomerApp to an API CustomerApp
func customerAppToAPI(a customerentity.CustomerApp) (api.CustomerApp, error) {
	apiCustomerApp := api.CustomerApp{}

	switch customerApp := a.Data.(type) {
	case appstripeentity.CustomerAppData:

		apiStripeCustomerApp := api.StripeCustomerApp{
			Id:   &a.AppID.ID,
			Type: api.StripeCustomerAppTypeStripe,
			Data: api.StripeCustomerAppData{
				StripeCustomerId:             customerApp.StripeCustomerID,
				StripeDefaultPaymentMethodId: customerApp.StripeDefaultPaymentMethodID,
			},
		}

		err := apiCustomerApp.FromStripeCustomerApp(apiStripeCustomerApp)
		if err != nil {
			return apiCustomerApp, fmt.Errorf("error converting to stripe customer app: %w", err)
		}

	default:
		return apiCustomerApp, fmt.Errorf("unsupported customer app data type: %s", a.Type)
	}

	return apiCustomerApp, nil
}
