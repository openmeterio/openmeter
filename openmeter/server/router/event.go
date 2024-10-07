package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// 32 days
const maximumFromDuration = time.Hour * 24 * 32

func (a *Router) IngestEvents(w http.ResponseWriter, r *http.Request) {
	a.config.IngestHandler.ServeHTTP(w, r)
}

func (a *Router) ListEvents(w http.ResponseWriter, r *http.Request, params api.ListEventsParams) {
	ctx := contextx.WithAttr(r.Context(), "operation", "queryEvents")

	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	minimumFrom := time.Now().Add(-maximumFromDuration)

	// Set default values
	from := defaultx.WithDefault(params.From, minimumFrom)
	limit := defaultx.WithDefault(params.Limit, 100)

	// Validate params
	if from.Before(minimumFrom) {
		err := fmt.Errorf("from date is too old: %s", from)

		a.config.ErrorHandler.HandleContext(ctx, errorsx.NewWarnError(err))
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w)

		return
	}

	if params.To != nil && params.To.Before(from) {
		err := fmt.Errorf("to date is before from date: %s < %s", params.To, params.From)

		a.config.ErrorHandler.HandleContext(ctx, errorsx.NewWarnError(err))
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w)

		return
	}

	if params.IngestedAtFrom != nil && params.IngestedAtFrom.Before(minimumFrom) {
		err := fmt.Errorf("ingestedAtFrom date is too old: %s", params.IngestedAtFrom)

		a.config.ErrorHandler.HandleContext(ctx, errorsx.NewWarnError(err))
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w)

		return
	}

	if params.IngestedAtFrom != nil && params.IngestedAtTo != nil && params.IngestedAtTo.Before(*params.IngestedAtFrom) {
		err := fmt.Errorf("ingestedAtTo date is before ingestedAtFrom date: %s < %s", params.IngestedAtTo, params.IngestedAtFrom)

		a.config.ErrorHandler.HandleContext(ctx, errorsx.NewWarnError(err))
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w)

		return
	}

	queryParams := streaming.ListEventsParams{
		From:           &from,
		To:             params.To,
		IngestedAtFrom: params.IngestedAtFrom,
		IngestedAtTo:   params.IngestedAtTo,
		ID:             params.Id,
		Subject:        params.Subject,
		HasError:       params.HasError,
		Limit:          limit,
	}

	// Query events
	events, err := a.config.StreamingConnector.ListEvents(ctx, namespace, queryParams)
	if err != nil {
		err := fmt.Errorf("query events: %w", err)

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w)

		return
	}

	render.JSON(w, r, events)
}
