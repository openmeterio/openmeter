package router

import "net/http"

// List supported currencies
// (GET /api/v1/info/currencies)
func (a *Router) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	a.infoHandler.ListCurrencies().ServeHTTP(w, r)
}
