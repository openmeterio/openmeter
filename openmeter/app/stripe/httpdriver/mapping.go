package httpdriver

import (
	apiv1models "github.com/openmeterio/openmeter/api/models"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
)

// toAPIStripePortalSession maps a StripePortalSession to an API StripePortalSession
func toAPIStripePortalSession(portalSession appstripe.StripePortalSession) apiv1models.StripeCustomerPortalSession {
	apiPortalSession := apiv1models.StripeCustomerPortalSession{
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
func fromAPIAppStripeCustomerDataBase(apiStripeCustomerData apiv1models.StripeCustomerAppDataBase) appstripe.CustomerData {
	return appstripe.CustomerData{
		StripeCustomerID:             apiStripeCustomerData.StripeCustomerId,
		StripeDefaultPaymentMethodID: apiStripeCustomerData.StripeDefaultPaymentMethodId,
	}
}
