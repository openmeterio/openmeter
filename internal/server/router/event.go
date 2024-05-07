package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

func (a *Router) IngestEvents(w http.ResponseWriter, r *http.Request) {
	a.config.IngestHandler.ServeHTTP(w, r)
}

func (a *Router) ListEvents(w http.ResponseWriter, r *http.Request, _ api.ListEventsParams) {
	a.listEventsHandler.ServeHTTP(w, r)
}
