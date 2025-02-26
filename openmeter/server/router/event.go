package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// POST /api/v1/events
func (a *Router) IngestEvents(w http.ResponseWriter, r *http.Request) {
	a.config.IngestHandler.ServeHTTP(w, r)
}

// GET /api/v1/events
func (a *Router) ListEvents(w http.ResponseWriter, r *http.Request, params api.ListEventsParams) {
	a.meterEventHandler.ListEvents().With(params).ServeHTTP(w, r)
}
