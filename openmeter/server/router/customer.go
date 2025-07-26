package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	apphttpdriver "github.com/openmeterio/openmeter/openmeter/app/httpdriver"
	customerhttpdriver "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	subscriptionhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/http"
)

// List customers
// (GET /api/v1/customer/customers)
func (a *Router) ListCustomers(w http.ResponseWriter, r *http.Request, params api.ListCustomersParams) {
	a.customerHandler.ListCustomers().With(params).ServeHTTP(w, r)
}

// Create a customer
// (POST /api/v1/customer/customers)
func (a *Router) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	a.customerHandler.CreateCustomer().ServeHTTP(w, r)
}

// Delete a customer
// (DELETE /api/v1/customer/customers/{customerId})
func (a *Router) DeleteCustomer(w http.ResponseWriter, r *http.Request, customerIDOrKey string) {
	a.customerHandler.DeleteCustomer().With(customerIDOrKey).ServeHTTP(w, r)
}

// Get customer
// (GET /api/v1/customer/customers/{customerId})
func (a *Router) GetCustomer(w http.ResponseWriter, r *http.Request, customerIDOrKey string, params api.GetCustomerParams) {
	a.customerHandler.GetCustomer().With(customerhttpdriver.GetCustomerParams{
		CustomerIDOrKey:   customerIDOrKey,
		GetCustomerParams: params,
	}).ServeHTTP(w, r)
}

// Update customer
// (PUT /api/v1/customer/customers/{customerId})
func (a *Router) UpdateCustomer(w http.ResponseWriter, r *http.Request, customerIDOrKey string) {
	a.customerHandler.UpdateCustomer().With(customerIDOrKey).ServeHTTP(w, r)
}

// List customer apps
// (GET /api/v1/customer/customers/{customerId}/apps)
func (a *Router) ListCustomerAppData(w http.ResponseWriter, r *http.Request, customerIdOrKey string, params api.ListCustomerAppDataParams) {
	a.appHandler.ListCustomerData().With(apphttpdriver.ListCustomerDataParams{
		ListCustomerAppDataParams: params,
		CustomerIdOrKey:           customerIdOrKey,
	}).ServeHTTP(w, r)
}

// Upsert customer app data
// (PUT /api/v1/customer/customers/{customerId}/apps/{appId})
func (a *Router) UpsertCustomerAppData(w http.ResponseWriter, r *http.Request, customerIDOrKey string) {
	a.appHandler.UpsertCustomerData().With(apphttpdriver.UpsertCustomerDataParams{
		CustomerIdOrKey: customerIDOrKey,
	}).ServeHTTP(w, r)
}

// Delete customer app data
// (DELETE /api/v1/customer/customers/{customerId}/apps/{appId})
func (a *Router) DeleteCustomerAppData(w http.ResponseWriter, r *http.Request, customerIDOrKey string, appID string) {
	a.appHandler.DeleteCustomerData().With(apphttpdriver.DeleteCustomerDataParams{
		CustomerIdOrKey: customerIDOrKey,
		AppId:           appID,
	}).ServeHTTP(w, r)
}

// List customer subscriptions
// (GET /api/v1/customer/customers/{customerId}/subscriptions)
func (a *Router) ListCustomerSubscriptions(w http.ResponseWriter, r *http.Request, customerIDOrKey string, params api.ListCustomerSubscriptionsParams) {
	a.subscriptionHandler.ListCustomerSubscriptions().With(subscriptionhttpdriver.ListCustomerSubscriptionsParams{
		CustomerIDOrKey: customerIDOrKey,
		Params:          params,
	}).ServeHTTP(w, r)
}

// Get customer stripe app data
// (GET /api/v1/customers/{customerIdOrKey}/stripe)
func (a *Router) GetCustomerStripeAppData(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	a.appHandler.GetCustomerStripeAppData().With(apphttpdriver.GetCustomerStripeAppDataParams{
		CustomerIdOrKey: customerIdOrKey,
	}).ServeHTTP(w, r)
}

// Upsert customer stripe app data
// (PUT /api/v1/customers/{customerIdOrKey}/stripe)
func (a *Router) UpsertCustomerStripeAppData(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	a.appHandler.UpsertCustomerStripeAppData().With(apphttpdriver.UpsertCustomerStripeAppDataParams{
		CustomerIdOrKey: customerIdOrKey,
	}).ServeHTTP(w, r)
}

// Create Stripe customer portal session
// (POST /api/v1/customers/{customerIdOrKey}/stripe/portal)
func (a *Router) CreateCustomerStripePortalSession(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	unimplemented.CreateCustomerStripePortalSession(w, r, customerIdOrKey)
}
