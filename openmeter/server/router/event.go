package router

import (
	"net/http"

	apiv1models "github.com/openmeterio/openmeter/api/models"
)

// POST /api/v1/events
func (a *Router) IngestEvents(w http.ResponseWriter, r *http.Request) {
	a.ingestHandler.IngestEvents().ServeHTTP(w, r)
}

// GET /api/v1/events
func (a *Router) ListEvents(w http.ResponseWriter, r *http.Request, params apiv1models.ListEventsParams) {
	a.meterEventHandler.ListEvents().With(params).ServeHTTP(w, r)
}

// GET /api/v2/events
func (a *Router) ListEventsV2(w http.ResponseWriter, r *http.Request, params apiv1models.ListEventsV2Params) {
	a.meterEventHandler.ListEventsV2().With(params).ServeHTTP(w, r)
}
