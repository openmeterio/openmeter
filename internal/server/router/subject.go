package router

import (
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *Router) UpsertSubject(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "upsertSubject")

	err := fmt.Errorf("not implemented: subject management is only available in OpenMeter Cloud")

	// TODO: caller error, no need to pass to error handler
	a.config.ErrorHandler.HandleContext(ctx, err)
	models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) GetSubject(w http.ResponseWriter, r *http.Request, idOrKey string) {
	ctx := contextx.WithAttr(r.Context(), "operation", "getSubject")
	ctx = contextx.WithAttr(ctx, "id", idOrKey)

	err := fmt.Errorf("not implemented: subjects are only available in OpenMeter Cloud")

	// TODO: caller error, no need to pass to error handler
	a.config.ErrorHandler.HandleContext(ctx, err)
	models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) ListSubjects(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "listSubjects")
	err := fmt.Errorf("not implemented: subjects are only available in OpenMeter Cloud")

	// TODO: caller error, no need to pass to error handler
	a.config.ErrorHandler.HandleContext(ctx, err)
	models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) DeleteSubject(w http.ResponseWriter, r *http.Request, idOrKey string) {
	ctx := contextx.WithAttr(r.Context(), "operation", "deleteSubject")
	ctx = contextx.WithAttr(ctx, "id", idOrKey)

	err := fmt.Errorf("not implemented: subjects are only available in OpenMeter Cloud")

	// TODO: caller error, no need to pass to error handler
	a.config.ErrorHandler.HandleContext(ctx, err)
	models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w, r)
}
