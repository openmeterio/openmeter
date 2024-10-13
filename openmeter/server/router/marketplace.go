package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// List marketplace listings
// (GET /api/v1/marketplace/listings)
func (a *Router) MarketplaceListListings(w http.ResponseWriter, r *http.Request, params api.MarketplaceListListingsParams) {
	a.appHandler.MarketplaceListListings().With(params).ServeHTTP(w, r)
}

// Get a marketplace listing
// (GET /api/v1/marketplace/listings/{appType})
func (a *Router) GetMarketplaceListing(w http.ResponseWriter, r *http.Request, appType api.OpenMeterAppType) {
	a.appHandler.GetMarketplaceListing().With(appType).ServeHTTP(w, r)
}

// Install a marketplace app by API key
// (POST /api/v1/marketplace/install/apikey)
func (a *Router) MarketplaceAppAPIKeyInstall(w http.ResponseWriter, r *http.Request, appType api.OpenMeterAppType) {
	a.appHandler.MarketplaceAppAPIKeyInstall().With(appType).ServeHTTP(w, r)
}

// Get the URL to install a marketplace app by OAuth2
// (GET /api/v1/marketplace/install/oauth2)
func (a *Router) MarketplaceOAuth2InstallGetURL(w http.ResponseWriter, r *http.Request, appType api.OpenMeterAppType) {
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte("Unimplemented"))
}

// Authorize the installation of a marketplace app by OAuth2
// (// (GET /api/v1/marketplace/install/oauth2/authorize))
func (a *Router) MarketplaceOAuth2InstallAuthorize(w http.ResponseWriter, r *http.Request, appType api.OpenMeterAppType, params api.MarketplaceOAuth2InstallAuthorizeParams) {
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte("Unimplemented"))
}
