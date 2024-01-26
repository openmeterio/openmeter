package router

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (a *Router) ListMeters(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("operation", "listMeters")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	meters, err := a.config.Meters.ListMeters(r.Context(), namespace)
	if err != nil {
		err := fmt.Errorf("list meters: %w", err)
		errorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError), w, r)
		return
	}

	// TODO: remove once meter model pointer is removed
	list := slicesx.Map[models.Meter, render.Renderer](meters, func(meter models.Meter) render.Renderer {
		return &meter
	})

	_ = render.RenderList(w, r, list)
}

func (a *Router) CreateMeter(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("operation", "createMeter")
	err := fmt.Errorf("not implemented: manage meters via config or checkout OpenMeter Cloud")
	errorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented), w, r)
}

func (a *Router) DeleteMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string) {
	logger := slog.With("operation", "deleteMeter", "id", meterIdOrSlug)
	err := fmt.Errorf("not implemented: manage meters via config or checkout OpenMeter Cloud")
	errorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented), w, r)
}

func (a *Router) GetMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string) {
	logger := slog.With("operation", "getMeter", "id", meterIdOrSlug)
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	meter, err := a.config.Meters.GetMeterByIDOrSlug(r.Context(), namespace, meterIdOrSlug)

	// TODO: remove once meter model pointer is removed
	if e := (&models.MeterNotFoundError{}); errors.As(err, &e) {
		errorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusNotFound), w, r)
		return
	} else if err != nil {
		err := fmt.Errorf("get meter: %w", err)
		errorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError), w, r)
		return
	}

	// TODO: remove once meter model pointer is removed
	_ = render.Render(w, r, &meter)
}
