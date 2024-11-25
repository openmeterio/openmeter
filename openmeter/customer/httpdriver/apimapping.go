package httpdriver

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
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

func mapApp(namespace string, apiApp api.CustomerApp) customerentity.CustomerApp {
	customerApp := customerentity.CustomerApp{
		Type: appentitybase.AppType(apiApp.Type),
		Data: apiApp.Data,
	}

	if apiApp.Id != nil {
		customerApp.AppID = &appentitybase.AppID{
			Namespace: namespace,
			ID:        *apiApp.Id,
		}
	}

	return customerApp
}
