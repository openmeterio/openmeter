package router

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *Router) IngestEvents(w http.ResponseWriter, r *http.Request) {
	// TODO: add error handling

	// namespace := a.config.NamespaceManager.GetDefaultNamespace()
	a.config.IngestHandler.ServeHTTP(w, r)
}

func (a *Router) ListEvents(w http.ResponseWriter, r *http.Request, params api.ListEventsParams) {
	ctx := contextx.WithAttr(r.Context(), "operation", "queryEvents")

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

	events, err := a.config.StreamingConnector.ListEvents(ctx, namespace, queryParams)
	if err != nil {
		err := fmt.Errorf("query events: %w", err)

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)

		return

	}

	render.JSON(w, r, events)
}
