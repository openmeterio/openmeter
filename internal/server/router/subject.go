package router

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *Router) UpsertSubject(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("operation", "upsertSubject")
	err := fmt.Errorf("not implemented: subject management is only available in OpenMeter Cloud")
	errorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented), w, r)
}

func (a *Router) GetSubject(w http.ResponseWriter, r *http.Request, idOrKey string) {
	logger := slog.With("operation", "getSubject", "id", idOrKey)
	err := fmt.Errorf("not implemented: subjects are only available in OpenMeter Cloud")
	errorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented), w, r)
}

func (a *Router) ListSubjects(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("operation", "listSubjects")
	err := fmt.Errorf("not implemented: subjects are only available in OpenMeter Cloud")
	errorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented), w, r)
}

func (a *Router) DeleteSubject(w http.ResponseWriter, r *http.Request, idOrKey string) {
	logger := slog.With("operation", "deleteSubject", "id", idOrKey)
	err := fmt.Errorf("not implemented: subjects are only available in OpenMeter Cloud")
	errorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented), w, r)
}
