package router

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ListMeterSubjects lists the subjects of a meter.
func (a *Router) ListMeterSubjects(w http.ResponseWriter, r *http.Request, meterIDOrSlug string) {
	ctx := contextx.WithAttr(r.Context(), "operation", "listMeterSubjects")
	ctx = contextx.WithAttr(ctx, "id", meterIDOrSlug)

	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Get meter
	meter, err := a.config.Meters.GetMeterByIDOrSlug(ctx, namespace, meterIDOrSlug)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			err := fmt.Errorf("meter not found: %w", err)

			models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w)

			return
		}

		err := fmt.Errorf("get meter: %w", err)

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w)

		return
	}

	subjects, err := a.config.StreamingConnector.ListMeterSubjects(ctx, namespace, meter, streaming.ListMeterSubjectsParams{})
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			err := fmt.Errorf("meter not found: %w", err)

			models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w)

			return
		}

		err := fmt.Errorf("list meter subjects: %w", err)

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w)

		return
	}

	render.JSON(w, r, subjects)
}
