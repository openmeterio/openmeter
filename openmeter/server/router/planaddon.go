package router

import "net/http"

// List all available add-ons for plan
// (GET /api/v1/plans/{planId}/addons)
func (a *Router) ListPlanAddons(w http.ResponseWriter, r *http.Request, planId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create new add-on assignment for plan
// (POST /api/v1/plans/{planId}/addons)
func (a *Router) CreatePlanAddon(w http.ResponseWriter, r *http.Request, planId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Delete add-on assignment for plan
// (DELETE /api/v1/plans/{planId}/addons/{planAddonId})
func (a *Router) DeletePlanAddon(w http.ResponseWriter, r *http.Request, planId string, planAddonId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get add-on assignment for plan
// (GET /api/v1/plans/{planId}/addons/{planAddonId})
func (a *Router) GetPlanAddon(w http.ResponseWriter, r *http.Request, planId string, planAddonId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Update add-on assignment for plan
// (PUT /api/v1/plans/{planId}/addons/{planAddonId})
func (a *Router) UpdatePlanAddon(w http.ResponseWriter, r *http.Request, planId string, planAddonId string) {
	w.WriteHeader(http.StatusNotImplemented)
}
