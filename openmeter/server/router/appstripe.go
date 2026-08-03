package router

import (
	"net/http"

	appstripehttpdriver "github.com/openmeterio/openmeter/openmeter/app/stripe/httpdriver"
)

// Handle app stripe webhook
// (POST /api/v1/apps/{appId}/stripe/webhook)
func (a *Router) AppStripeWebhook(w http.ResponseWriter, r *http.Request, appID string) {
	a.appStripeHandler.AppStripeWebhook().With(appstripehttpdriver.AppStripeWebhookParams{
		AppID: appID,
	}).ServeHTTP(w, r)
}

// Handle update stripe api key
// (POST /api/v1/apps/{id}/stripe/api-key)
func (a *Router) UpdateStripeAPIKey(w http.ResponseWriter, r *http.Request, appID string) {
	a.appStripeHandler.UpdateStripeAPIKey().With(appID).ServeHTTP(w, r)
}

// Handle create app stripe checkout session
// (POST /api/v1/stripe/checkout/sessions)
func (a *Router) CreateStripeCheckoutSession(w http.ResponseWriter, r *http.Request) {
	a.appStripeHandler.CreateAppStripeCheckoutSession().ServeHTTP(w, r)
}
