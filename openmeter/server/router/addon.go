package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// List addons
// (GET /api/v1/addons)
func (a *Router) ListAddons(w http.ResponseWriter, r *http.Request, params api.ListAddonsParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create a addon
// (POST /api/v1/addons)
func (a *Router) CreateAddon(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Delete addon
// (DELETE /api/v1/addons/{addonId})
func (a *Router) DeleteAddon(w http.ResponseWriter, r *http.Request, addonId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get addon
// (GET /api/v1/addons/{addonIdOrKey})
func (a *Router) GetAddon(w http.ResponseWriter, r *http.Request, addonIdOrKey string, params api.GetAddonParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Update a addon
// (PUT /api/v1/addons/{addonId})
func (a *Router) UpdateAddon(w http.ResponseWriter, r *http.Request, addonId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Archive addon version
// (POST /api/v1/addons/{addonId}/archive)
func (a *Router) ArchiveAddon(w http.ResponseWriter, r *http.Request, addonId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Publish addon
// (POST /api/v1/addons/{addonId}/publish)
func (a *Router) PublishAddon(w http.ResponseWriter, r *http.Request, addonId string) {
	w.WriteHeader(http.StatusNotImplemented)
}
