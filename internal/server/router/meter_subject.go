package router

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ListMeterSubjects lists the subjects of a meter.
func (a *Router) ListMeterSubjects(w http.ResponseWriter, r *http.Request, meterIDOrSlug string) {
	logger := slog.With("operation", "listMeterSubjects", "id", meterIDOrSlug)
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	subjects, err := a.config.StreamingConnector.ListMeterSubjects(r.Context(), namespace, meterIDOrSlug, nil, nil)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			err := fmt.Errorf("meter not found: %w", err)
			models.NewStatusProblem(r.Context(), err, http.StatusNotFound).Respond(logger, w, r)
			return
		}

		err := fmt.Errorf("list meter subjects: %w", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(logger, w, r)
		return
	}

	render.JSON(w, r, subjects)
}
