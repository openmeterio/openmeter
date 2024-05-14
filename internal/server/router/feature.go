package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// Get feature, GET:/api/v1/features/{featureID}
func (a *Router) GetFeature(w http.ResponseWriter, r *http.Request, featureID api.FeatureID) {
	a.CreditHandler.GetFeature(r.Context(), w, r, featureID)
	// TODO: remove chi renderers
}

// List features: GET /api/v1/features
func (a *Router) ListFeatures(w http.ResponseWriter, r *http.Request) {
	a.CreditHandler.ListFeatures(r.Context(), w, r)
}

// Create feature, POST: /api/v1/features
func (a *Router) CreateFeature(w http.ResponseWriter, r *http.Request) {
	a.CreditHandler.CreateFeature(r.Context(), w, r)
}

// Delete feature, DELETE:/api/v1/features/{featureID}
func (a *Router) DeleteFeature(w http.ResponseWriter, r *http.Request, featureID api.FeatureID) {
	a.CreditHandler.DeleteFeature(r.Context(), w, r, featureID)
}
