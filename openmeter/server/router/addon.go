package router

import (
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	addonhttpdriver "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/httpdriver"
)

// List addons
// (GET /api/v1/addons)
func (a *Router) ListAddons(w http.ResponseWriter, r *http.Request, params api.ListAddonsParams) {
	a.addonHandler.ListAddons().With(params).ServeHTTP(w, r)
}

// Create a addon
// (POST /api/v1/addons)
func (a *Router) CreateAddon(w http.ResponseWriter, r *http.Request) {
	a.addonHandler.CreateAddon().ServeHTTP(w, r)
}

// Delete addon
// (DELETE /api/v1/addons/{addonId})
func (a *Router) DeleteAddon(w http.ResponseWriter, r *http.Request, addonId string) {
	a.addonHandler.DeleteAddon().With(addonId).ServeHTTP(w, r)
}

// Get addon
// (GET /api/v1/addons/{addonIdOrKey})
func (a *Router) GetAddon(w http.ResponseWriter, r *http.Request, addonIdOrKey string, params api.GetAddonParams) {
	a.addonHandler.GetAddon().With(addonhttpdriver.GetAddonRequestParams{
		IDOrKey:       addonIdOrKey,
		IncludeLatest: lo.FromPtrOr(params.IncludeLatest, false),
	}).ServeHTTP(w, r)
}

// Update a addon
// (PUT /api/v1/addons/{addonId})
func (a *Router) UpdateAddon(w http.ResponseWriter, r *http.Request, addonId string) {
	a.addonHandler.UpdateAddon().With(addonId).ServeHTTP(w, r)
}

// Archive addon version
// (POST /api/v1/addons/{addonId}/archive)
func (a *Router) ArchiveAddon(w http.ResponseWriter, r *http.Request, addonId string) {
	a.addonHandler.ArchiveAddon().With(addonId).ServeHTTP(w, r)
}

// Publish addon
// (POST /api/v1/addons/{addonId}/publish)
func (a *Router) PublishAddon(w http.ResponseWriter, r *http.Request, addonId string) {
	a.addonHandler.PublishAddon().With(addonId).ServeHTTP(w, r)
}
