package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// Get feature, GET:/api/v1/features/{featureID}
func (a *Router) GetFeature(w http.ResponseWriter, r *http.Request, featureID api.FeatureID) {
	a.CreditHandlers.GetFeature.With(featureID).ServeHTTP(w, r)
	// TODO: remove chi renderers
}

// List features: GET /api/v1/features
func (a *Router) ListFeatures(w http.ResponseWriter, r *http.Request) {
	a.CreditHandlers.ListFeatures.ServeHTTP(w, r)
}

// Create feature, POST: /api/v1/features
func (a *Router) CreateFeature(w http.ResponseWriter, r *http.Request) {
	a.CreditHandlers.CreateFeature.ServeHTTP(w, r)
}

// Delete feature, DELETE:/api/v1/features/{featureID}
func (a *Router) DeleteFeature(w http.ResponseWriter, r *http.Request, featureID api.FeatureID) {
	a.CreditHandlers.DeleteFeature.With(featureID).ServeHTTP(w, r)
}
