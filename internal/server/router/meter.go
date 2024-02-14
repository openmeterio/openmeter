package router

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *Router) ListMeters(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "listMeters")

	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	meters, err := a.config.Meters.ListMeters(r.Context(), namespace)
	if err != nil {
		err := fmt.Errorf("list meters: %w", err)

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)

		return
	}

	// TODO: remove once meter model pointer is removed
	list := slicesx.Map[models.Meter, render.Renderer](meters, func(meter models.Meter) render.Renderer {
		return &meter
	})

	_ = render.RenderList(w, r, list)
}

func (a *Router) CreateMeter(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "createMeter")

	err := fmt.Errorf("not implemented: manage meters via config or checkout OpenMeter Cloud")

	// TODO: caller error, no need to pass to error handler
	a.config.ErrorHandler.HandleContext(ctx, err)
	models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) DeleteMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string) {
	ctx := contextx.WithAttr(r.Context(), "operation", "deleteMeter")
	ctx = contextx.WithAttr(ctx, "id", meterIdOrSlug)

	err := fmt.Errorf("not implemented: manage meters via config or checkout OpenMeter Cloud")

	a.config.ErrorHandler.HandleContext(ctx, err)
	models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) GetMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string) {
	ctx := contextx.WithAttr(r.Context(), "operation", "getMeter")
	ctx = contextx.WithAttr(ctx, "id", meterIdOrSlug)

	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	meter, err := a.config.Meters.GetMeterByIDOrSlug(ctx, namespace, meterIdOrSlug)

	// TODO: remove once meter model pointer is removed
	if e := (&models.MeterNotFoundError{}); errors.As(err, &e) {
		// TODO: caller error, no need to pass to error handler
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w, r)

		return
	} else if err != nil {
		err := fmt.Errorf("get meter: %w", err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)

		return
	}

	// TODO: remove once meter model pointer is removed
	_ = render.Render(w, r, &meter)
}
