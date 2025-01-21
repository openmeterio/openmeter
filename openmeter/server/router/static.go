package router

import "net/http"

// List supported currencies
// (GET /api/v1/static/currencies)
func (a *Router) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	a.staticHandler.ListCurrencies().ServeHTTP(w, r)
}
