package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/debug/httpdriver"
)

// Get debug metrics
// (GET /api/v1/debug/metrics)
func (a *Router) GetDebugMetrics(w http.ResponseWriter, r *http.Request) {
	a.debugHandler.GetMetrics().With(httpdriver.GetMetricsHandlerParams{}).ServeHTTP(w, r)
}
