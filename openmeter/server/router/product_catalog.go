package router

import (
	"net/http"

	apiv1models "github.com/openmeterio/openmeter/api/models"
)

// List features
// (GET /api/v1/features)
func (a *Router) ListFeatures(w http.ResponseWriter, r *http.Request, params apiv1models.ListFeaturesParams) {
	a.featureHandler.ListFeatures().With(params).ServeHTTP(w, r)
}

// Create feature
// (POST /api/v1/features)
func (a *Router) CreateFeature(w http.ResponseWriter, r *http.Request) {
	a.featureHandler.CreateFeature().ServeHTTP(w, r)
}

// Delete feature
// (DELETE /api/v1/features/{featureId})
func (a *Router) DeleteFeature(w http.ResponseWriter, r *http.Request, featureId string) {
	a.featureHandler.DeleteFeature().With(featureId).ServeHTTP(w, r)
}

// Get feature
// (GET /api/v1/features/{featureId})
func (a *Router) GetFeature(w http.ResponseWriter, r *http.Request, featureId string) {
	a.featureHandler.GetFeature().With(featureId).ServeHTTP(w, r)
}
