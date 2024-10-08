package router

import (
	"net/http"
)

// Create a customer
// (POST /api/v1/customer/customers)
func (a *Router) MarketplaceAPIKeyInstall(w http.ResponseWriter, r *http.Request) {
	a.appHandler.MarketplaceAppAPIKeyInstall().ServeHTTP(w, r)
}
