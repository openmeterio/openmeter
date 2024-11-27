package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app/httpdriver"
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
func (a *Router) DeleteCustomer(w http.ResponseWriter, r *http.Request, customerID string) {
	a.customerHandler.DeleteCustomer().With(customerID).ServeHTTP(w, r)
}

// Get customer
// (GET /api/v1/customer/customers/{customerId})
func (a *Router) GetCustomer(w http.ResponseWriter, r *http.Request, customerID string) {
	a.customerHandler.GetCustomer().With(customerID).ServeHTTP(w, r)
}

// Update customer
// (PUT /api/v1/customer/customers/{customerId})
func (a *Router) UpdateCustomer(w http.ResponseWriter, r *http.Request, customerID string) {
	a.customerHandler.UpdateCustomer().With(customerID).ServeHTTP(w, r)
}

// List customer apps
// (GET /api/v1/customer/customers/{customerId}/apps)
func (a *Router) ListCustomerAppData(w http.ResponseWriter, r *http.Request, customerID string, params api.ListCustomerAppDataParams) {
	a.appHandler.ListCustomerData().With(httpdriver.ListCustomerDataParams{
		ListCustomerAppDataParams: params,
		CustomerId:                customerID,
	}).ServeHTTP(w, r)
}

// Upsert customer app data
// (PUT /api/v1/customer/customers/{customerId}/apps/{appId})
func (a *Router) UpsertCustomerAppData(w http.ResponseWriter, r *http.Request, customerID string) {
	a.appHandler.UpsertCustomerData().With(httpdriver.UpsertCustomerDataParams{
		CustomerId: customerID,
	}).ServeHTTP(w, r)
}

// Delete customer app data
// (DELETE /api/v1/customer/customers/{customerId}/apps/{appId})
func (a *Router) DeleteCustomerAppData(w http.ResponseWriter, r *http.Request, customerID string, appID string) {
	a.appHandler.DeleteCustomerData().With(httpdriver.DeleteCustomerDataParams{
		CustomerId: customerID,
		AppId:      appID,
	}).ServeHTTP(w, r)
}
