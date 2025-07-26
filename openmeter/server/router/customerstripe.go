package router

import (
	"net/http"

	appstripehttpdriver "github.com/openmeterio/openmeter/openmeter/app/stripe/httpdriver"
)

// Get customer stripe app data
// (GET /api/v1/customers/{customerIdOrKey}/stripe)
func (a *Router) GetCustomerStripeAppData(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	a.appStripeHandler.GetCustomerStripeAppData().With(appstripehttpdriver.GetCustomerStripeAppDataParams{
		CustomerIdOrKey: customerIdOrKey,
	}).ServeHTTP(w, r)
}

// Upsert customer stripe app data
// (PUT /api/v1/customers/{customerIdOrKey}/stripe)
func (a *Router) UpsertCustomerStripeAppData(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	a.appStripeHandler.UpsertCustomerStripeAppData().With(appstripehttpdriver.UpsertCustomerStripeAppDataParams{
		CustomerIdOrKey: customerIdOrKey,
	}).ServeHTTP(w, r)
}

// Create Stripe customer portal session
// (POST /api/v1/customers/{customerIdOrKey}/stripe/portal)
func (a *Router) CreateCustomerStripePortalSession(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	a.appStripeHandler.CreateStripeCustomerPortalSession().With(appstripehttpdriver.CreateStripeCustomerPortalSessionParams{
		CustomerIdOrKey: customerIdOrKey,
	}).ServeHTTP(w, r)
}
