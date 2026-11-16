package httpdriver

import (
	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	appconvert "github.com/openmeterio/openmeter/openmeter/app/apiconvert"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
)

func MapAppToAPI(item app.App) (apiv1models.App, error) {
	return appconvert.ToAPIApp(item)
}

func MapEventAppToAPI(event app.EventApp) (apiv1models.App, error) {
	return appconvert.ToAPIEventApp(event)
}

// fromAPIAppStripeCustomerData maps an API stripe customer data to an app stripe customer data
func fromAPIAppStripeCustomerData(apiStripeCustomerData apiv1models.StripeCustomerAppData) appstripe.CustomerData {
	return appstripe.CustomerData{
		StripeCustomerID:             apiStripeCustomerData.StripeCustomerId,
		StripeDefaultPaymentMethodID: apiStripeCustomerData.StripeDefaultPaymentMethodId,
	}
}

// customerAppToAPI converts a CustomerApp to an API CustomerAppData
func ToAPIStripeCustomerAppData(
	customerAppData appstripe.CustomerData,
	stripeApp appstripe.App,
) apiv1models.StripeCustomerAppData {
	apiStripeCustomerAppData := apiv1models.StripeCustomerAppData{
		Id:                           lo.ToPtr(stripeApp.GetID().ID),
		Type:                         apiv1models.StripeCustomerAppDataTypeStripe,
		App:                          lo.ToPtr(appconvert.ToAPIStripeApp(stripeApp.Meta)),
		StripeCustomerId:             customerAppData.StripeCustomerID,
		StripeDefaultPaymentMethodId: customerAppData.StripeDefaultPaymentMethodID,
	}

	return apiStripeCustomerAppData
}
