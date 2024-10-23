package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
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
