package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// List features
// (GET /api/v1/features)
func (a *Router) ListFeatures(w http.ResponseWriter, r *http.Request, params api.ListFeaturesParams) {
	if !a.config.EntitlementsEnabled {
		unimplemented.ListFeatures(w, r, params)
		return
	}
	a.featureHandler.ListFeatures().With(params).ServeHTTP(w, r)
}

// Create feature
// (POST /api/v1/features)
func (a *Router) CreateFeature(w http.ResponseWriter, r *http.Request) {
	if !a.config.EntitlementsEnabled {
		unimplemented.CreateFeature(w, r)
		return
	}
	a.featureHandler.CreateFeature().ServeHTTP(w, r)
}

// Delete feature
// (DELETE /api/v1/features/{featureId})
func (a *Router) DeleteFeature(w http.ResponseWriter, r *http.Request, featureId api.FeatureId) {
	if !a.config.EntitlementsEnabled {
		unimplemented.DeleteFeature(w, r, featureId)
		return

	}
	a.featureHandler.DeleteFeature().With(featureId).ServeHTTP(w, r)
}

// Get feature
// (GET /api/v1/features/{featureId})
func (a *Router) GetFeature(w http.ResponseWriter, r *http.Request, featureId api.FeatureId) {
	if !a.config.EntitlementsEnabled {
		unimplemented.GetFeature(w, r, featureId)
		return
	}
	a.featureHandler.GetFeature().With(featureId).ServeHTTP(w, r)
}
