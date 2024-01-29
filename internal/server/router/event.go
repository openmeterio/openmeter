package router

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *Router) IngestEvents(w http.ResponseWriter, r *http.Request) {
	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	a.config.IngestHandler.ServeHTTP(w, r, namespace)
}

func (a *Router) ListEvents(w http.ResponseWriter, r *http.Request, params api.ListEventsParams) {
	logger := slog.With("operation", "queryEvents")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	limit := 100
	if params.Limit != nil {
		limit = *params.Limit
	}

	queryParams := streaming.ListEventsParams{
		From:  params.From,
		To:    params.To,
		Limit: limit,
	}

	events, err := a.config.StreamingConnector.ListEvents(r.Context(), namespace, queryParams)
	if err != nil {
		err := fmt.Errorf("query events: %w", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(logger, w, r)
		return

	}

	render.JSON(w, r, events)
}
