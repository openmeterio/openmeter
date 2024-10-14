package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// Handle to get app by id
// (GET /api/v1/apps/{appId})
func (a *Router) GetApp(w http.ResponseWriter, r *http.Request, appID string) {
	a.appHandler.GetApp().With(appID).ServeHTTP(w, r)
}

// Handle to list apps
// (GET /api/v1/apps)
func (a *Router) ListApps(w http.ResponseWriter, r *http.Request, params api.ListAppsParams) {
	a.appHandler.ListApps().With(params).ServeHTTP(w, r)
}

// Uninstall an app by id
// (DELETE /api/v1/apps/{id})
func (a *Router) UninstallApp(w http.ResponseWriter, r *http.Request, appID string) {
	a.appHandler.UninstallApp().With(appID).ServeHTTP(w, r)
}
