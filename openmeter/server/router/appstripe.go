package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// Handle app stripe webhook
// (POST /api/v1/apps/{appId}/stripe/webhook)
func (a *Router) AppStripeWebhook(w http.ResponseWriter, r *http.Request, appID api.ULID) {
	a.appStripeHandler.AppStripeWebhook().With(appID).ServeHTTP(w, r)
}

// Handle create app stripe checkout session
// (POST /api/v1/apps/{appId}/stripe/checkout-session)
func (a *Router) CreateAppStripeCheckoutSession(w http.ResponseWriter, r *http.Request, appID api.ULID) {
	a.appStripeHandler.CreateAppStripeCheckoutSession().With(appID).ServeHTTP(w, r)
}
