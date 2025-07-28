package httpdriver

import (
	"github.com/openmeterio/openmeter/api"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
)

// toAPIStripePortalSession maps a StripePortalSession to an API StripePortalSession
func toAPIStripePortalSession(portalSession appstripeentity.StripePortalSession) api.StripeCustomerPortalSession {
	apiPortalSession := api.StripeCustomerPortalSession{
		Id:               portalSession.ID,
		StripeCustomerId: portalSession.StripeCustomerID,
		ReturnUrl:        portalSession.ReturnURL,
		Url:              portalSession.URL,
		CreatedAt:        portalSession.CreatedAt,
		Livemode:         portalSession.Livemode,
		Locale:           portalSession.Locale,
	}

	if portalSession.Configuration != nil {
		apiPortalSession.ConfigurationId = portalSession.Configuration.ID
	}

	return apiPortalSession
}

// fromAPIAppStripeCustomerDataBase maps an API stripe customer data base to an app stripe customer data
func fromAPIAppStripeCustomerDataBase(apiStripeCustomerData api.StripeCustomerAppDataBase) appstripeentity.CustomerData {
	return appstripeentity.CustomerData{
		StripeCustomerID:             apiStripeCustomerData.StripeCustomerId,
		StripeDefaultPaymentMethodID: apiStripeCustomerData.StripeDefaultPaymentMethodId,
	}
}
