package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/httpdriver"
)

// List all available add-ons for plan
// (GET /api/v1/plans/{planId}/addons)
func (a *Router) ListPlanAddons(w http.ResponseWriter, r *http.Request, planId string, params api.ListPlanAddonsParams) {
	a.planAddonHandler.ListPlanAddons().With(httpdriver.ListPlanAddonsParams{
		ListPlanAddonsParams: params,
		PlanIDOrKey:          planId,
	}).ServeHTTP(w, r)
}

// Create new add-on assignment for plan
// (POST /api/v1/plans/{planId}/addons)
func (a *Router) CreatePlanAddon(w http.ResponseWriter, r *http.Request, planId string) {
	a.planAddonHandler.CreatePlanAddon().With(planId).ServeHTTP(w, r)
}

// Delete add-on assignment for plan
// (DELETE /api/v1/plans/{planId}/addons/{planAddonId})
func (a *Router) DeletePlanAddon(w http.ResponseWriter, r *http.Request, planId string, planAddonId string) {
	a.planAddonHandler.DeletePlanAddon().With(httpdriver.DeletePlanAddonParams{
		PlanID:  planId,
		AddonID: planAddonId,
	}).ServeHTTP(w, r)
}

// Get add-on assignment for plan
// (GET /api/v1/plans/{planId}/addons/{planAddonId})
func (a *Router) GetPlanAddon(w http.ResponseWriter, r *http.Request, planId string, planAddonId string) {
	a.planAddonHandler.GetPlanAddon().With(httpdriver.GetPlanAddonParams{
		PlanIDOrKey:  planId,
		AddonIDOrKey: planAddonId,
	}).ServeHTTP(w, r)
}

// Update add-on assignment for plan
// (PUT /api/v1/plans/{planId}/addons/{planAddonId})
func (a *Router) UpdatePlanAddon(w http.ResponseWriter, r *http.Request, planId string, planAddonId string) {
	a.planAddonHandler.UpdatePlanAddon().With(httpdriver.UpdatePlanAddonParams{
		PlanID:  planId,
		AddonID: planAddonId,
	}).ServeHTTP(w, r)
}
