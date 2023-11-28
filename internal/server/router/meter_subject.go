package router

import (
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
			logger.Warn("meter not found", "error", err)
			models.NewStatusProblem(r.Context(), err, http.StatusNotFound).Respond(w, r)
			return
		}

		logger.Error("connector", "error", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	render.JSON(w, r, subjects)
}
