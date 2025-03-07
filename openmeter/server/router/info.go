package router

import "net/http"

// List supported currencies
// (GET /api/v1/info/currencies)
func (a *Router) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	a.infoHandler.ListCurrencies().ServeHTTP(w, r)
}

// Get progress
// (GET /api/v1/info/progress/{progress_id})
func (a *Router) GetProgress(w http.ResponseWriter, r *http.Request, progressID string) {
	a.progressHandler.GetProgress().With(progressID).ServeHTTP(w, r)
}
