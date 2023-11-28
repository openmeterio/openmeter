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
		logger.Error("listing meters", "error", err)

		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w, r)

		return
	}

	// TODO: remove once meter model pointer is removed
	list := slicesx.Map[models.Meter, render.Renderer](meters, func(meter models.Meter) render.Renderer {
		return &meter
	})

	_ = render.RenderList(w, r, list)
}

func (a *Router) CreateMeter(w http.ResponseWriter, r *http.Request) {
	err := fmt.Errorf("not implemented: manage meters via config or checkout OpenMeter Cloud")
	models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) DeleteMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string) {
	err := fmt.Errorf("not implemented: manage meters via config or checkout OpenMeter Cloud")
	models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented).Respond(w, r)
}

func (a *Router) GetMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug string) {
	logger := slog.With("operation", "getMeter", "id", meterIdOrSlug)
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	meter, err := a.config.Meters.GetMeterByIDOrSlug(r.Context(), namespace, meterIdOrSlug)

	// TODO: remove once meter model pointer is removed
	if e := (&models.MeterNotFoundError{}); errors.As(err, &e) {
		logger.Debug("meter not found")

		// TODO: add meter id or slug as detail
		models.NewStatusProblem(r.Context(), errors.New("meter not found"), http.StatusNotFound).Respond(w, r)

		return
	} else if err != nil {
		logger.Error("getting meter", slog.Any("error", err))

		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w, r)

		return
	}

	// TODO: remove once meter model pointer is removed
	_ = render.Render(w, r, &meter)
}
