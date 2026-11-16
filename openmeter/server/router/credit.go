package router

import (
	"net/http"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	creditdriver "github.com/openmeterio/openmeter/openmeter/credit/driver"
)

// List grants
// (GET /api/v1/grants)
func (a *Router) ListGrants(w http.ResponseWriter, r *http.Request, params apiv1models.ListGrantsParams) {
	a.creditHandler.ListGrants().With(creditdriver.ListGrantsHandlerParams{
		Params: params,
	}).ServeHTTP(w, r)
}

// Delete a grant
// (DELETE /api/v1/grants/{grantId})
func (a *Router) VoidGrant(w http.ResponseWriter, r *http.Request, grantId string, params apiv1models.VoidGrantParams) {
	a.creditHandler.VoidGrant().With(creditdriver.VoidGrantHandlerParams{
		ID:     grantId,
		Params: params,
	}).ServeHTTP(w, r)
}

// ------------------------------------------------------------
// V2 APIS
// ------------------------------------------------------------

// List grants
// (GET /api/v2/grants)
func (a *Router) ListGrantsV2(w http.ResponseWriter, r *http.Request, params apiv1models.ListGrantsV2Params) {
	a.creditHandler.ListGrantsV2().With(creditdriver.ListGrantsV2HandlerParams{
		Params: params,
	}).ServeHTTP(w, r)
}
