package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	creditdriver "github.com/openmeterio/openmeter/openmeter/credit/driver"
)

// List grants
// (GET /api/v1/grants)
func (a *Router) ListGrants(w http.ResponseWriter, r *http.Request, params api.ListGrantsParams) {
	a.creditHandler.ListGrants().With(creditdriver.ListGrantsHandlerParams{
		Params: params,
	}).ServeHTTP(w, r)
}

// Delete a grant
// (DELETE /api/v1/grants/{grantId})
func (a *Router) VoidGrant(w http.ResponseWriter, r *http.Request, grantId string) {
	a.creditHandler.VoidGrant().With(creditdriver.VoidGrantHandlerParams{
		ID: grantId,
	}).ServeHTTP(w, r)
}

// ------------------------------------------------------------
// V2 APIS
// ------------------------------------------------------------

// List grants
// (GET /api/v2/grants)
func (a *Router) ListGrantsV2(w http.ResponseWriter, r *http.Request, params api.ListGrantsV2Params) {
	a.creditHandler.ListGrantsV2().With(creditdriver.ListGrantsV2HandlerParams{
		Params: params,
	}).ServeHTTP(w, r)
}
