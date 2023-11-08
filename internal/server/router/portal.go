package router

import (
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *Router) CreatePortalToken(w http.ResponseWriter, r *http.Request) {
	err := fmt.Errorf("not implemented: portal is an OpenMeter Cloud only feature")
	models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) InvalidatePortalTokens(w http.ResponseWriter, r *http.Request) {
	err := fmt.Errorf("not implemented: portal is an OpenMeter Cloud only feature")
	models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented).Respond(w, r)
}
