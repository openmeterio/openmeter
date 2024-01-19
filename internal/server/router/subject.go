package router

import (
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *Router) UpsertSubject(w http.ResponseWriter, r *http.Request) {
	err := fmt.Errorf("not implemented: subject management is only available in OpenMeter Cloud")
	models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) GetSubject(w http.ResponseWriter, r *http.Request, idOrKey string) {
	err := fmt.Errorf("not implemented: subjects are only available in OpenMeter Cloud")
	models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) ListSubjects(w http.ResponseWriter, r *http.Request) {
	err := fmt.Errorf("not implemented: subjects are only available in OpenMeter Cloud")
	models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) DeleteSubject(w http.ResponseWriter, r *http.Request, idOrKey string) {
	err := fmt.Errorf("not implemented: subjects are only available in OpenMeter Cloud")
	models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented).Respond(w, r)
}
