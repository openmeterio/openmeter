package router

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// Get feature, GET:/api/v1/features/{featureID}
func (a *Router) GetFeature(w http.ResponseWriter, r *http.Request, featureID api.FeatureID) {
	ctx := contextx.WithAttr(r.Context(), "operation", "getFeature")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	feature, err := a.config.CreditConnector.GetFeature(ctx, namespace, featureID)
	if err != nil {
		if _, ok := err.(*credit.FeatureNotFoundError); ok {
			models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w, r)
			return
		}

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	_ = render.Render(w, r, feature)
}

// List features: GET /api/v1/features
func (a *Router) ListFeatures(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "listFeatures")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	features, err := a.config.CreditConnector.ListFeatures(ctx, namespace, credit.ListFeaturesParams{})
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}
	list := slicesx.Map[credit.Feature, render.Renderer](features, func(feature credit.Feature) render.Renderer {
		return &feature
	})

	_ = render.RenderList(w, r, list)
}

// Create feature, POST: /api/v1/features
func (a *Router) CreateFeature(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "createFeature")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Parse request body
	featureIn := &api.CreateFeatureJSONRequestBody{}
	if err := render.DecodeJSON(r.Body, featureIn); err != nil {
		err := fmt.Errorf("decode json: %w", err)

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)

		return
	}

	meter, err := a.config.Meters.GetMeterByIDOrSlug(ctx, namespace, featureIn.MeterSlug)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			err := fmt.Errorf("meter not found: %s", featureIn.MeterSlug)
			a.config.ErrorHandler.HandleContext(ctx, err)
			models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
			return
		}
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	if err := validateMeterAggregation(meter); err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
		return
	}

	// Let's make sure we are not allowing the ID to be specified externally
	featureIn.ID = nil

	featureOut, err := a.config.CreditConnector.CreateFeature(ctx, namespace, *featureIn)
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	render.Status(r, http.StatusCreated)
	_ = render.Render(w, r, featureOut)
}

func validateMeterAggregation(meter models.Meter) error {
	switch meter.Aggregation {
	case models.MeterAggregationCount, models.MeterAggregationUniqueCount, models.MeterAggregationSum:
		return nil
	}

	return fmt.Errorf("meter %s's aggregation is %s but features can only be created for SUM, COUNT, UNIQUE_COUNT meters",
		meter.Slug,
		meter.Aggregation,
	)
}

// Delete feature, DELETE:/api/v1/features/{featureID}
func (a *Router) DeleteFeature(w http.ResponseWriter, r *http.Request, featureID api.FeatureID) {
	ctx := contextx.WithAttr(r.Context(), "operation", "deleteFeature")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	_, err := a.config.CreditConnector.GetFeature(ctx, namespace, featureID)
	if err != nil {
		if _, ok := err.(*credit.FeatureNotFoundError); ok {
			models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w, r)
			return
		}

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	err = a.config.CreditConnector.DeleteFeature(ctx, namespace, featureID)
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
