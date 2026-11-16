package router

import (
	"net/http"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	apphttpdriver "github.com/openmeterio/openmeter/openmeter/app/httpdriver"
	customerhttpdriver "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	subscriptionhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/http"
)

// List customers
// (GET /api/v1/customer/customers)
func (a *Router) ListCustomers(w http.ResponseWriter, r *http.Request, params apiv1models.ListCustomersParams) {
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
func (a *Router) GetCustomer(w http.ResponseWriter, r *http.Request, customerIDOrKey string, params apiv1models.GetCustomerParams) {
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
func (a *Router) ListCustomerAppData(w http.ResponseWriter, r *http.Request, customerIdOrKey string, params apiv1models.ListCustomerAppDataParams) {
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
func (a *Router) ListCustomerSubscriptions(w http.ResponseWriter, r *http.Request, customerIDOrKey string, params apiv1models.ListCustomerSubscriptionsParams) {
	a.subscriptionHandler.ListCustomerSubscriptions().With(subscriptionhttpdriver.ListCustomerSubscriptionsParams{
		CustomerIDOrKey: customerIDOrKey,
		Params:          params,
	}).ServeHTTP(w, r)
}
