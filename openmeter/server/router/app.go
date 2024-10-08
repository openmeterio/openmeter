package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// Handle to get app by id
// (GET /api/v1/apps/{appId})
func (a *Router) GetApp(w http.ResponseWriter, r *http.Request, appID string) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Unimplemented"))
}

// Handle to list apps
// (GET /api/v1/apps)
func (a *Router) ListApps(w http.ResponseWriter, r *http.Request, params api.ListAppsParams) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Unimplemented"))
}

// Uninstall an app by id
// (DELETE /api/v1/apps/{id})
func (a *Router) UninstallApp(w http.ResponseWriter, r *http.Request, id string) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Unimplemented"))
}
