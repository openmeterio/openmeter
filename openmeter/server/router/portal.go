package router

import (
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/portal/authenticator"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// POST /api/v1/portal/tokens
func (a *Router) CreatePortalToken(w http.ResponseWriter, r *http.Request) {
	a.portalHandler.CreateToken().ServeHTTP(w, r)
}

// GET /api/v1/portal/tokens
func (a *Router) ListPortalTokens(w http.ResponseWriter, r *http.Request, params api.ListPortalTokensParams) {
	a.portalHandler.ListTokens().With(params).ServeHTTP(w, r)
}

// POST /api/v1/portal/tokens/invalidate
func (a *Router) InvalidatePortalTokens(w http.ResponseWriter, r *http.Request) {
	a.portalHandler.InvalidateToken().ServeHTTP(w, r)
}

// TODO: migrate to http handler
// GET /api/v1/portal/meters/{meterSlug}/query
func (a *Router) QueryPortalMeter(w http.ResponseWriter, r *http.Request, meterSlug string, params api.QueryPortalMeterParams) {
	ctx := contextx.WithAttr(r.Context(), "operation", "queryPortalMeter")
	ctx = contextx.WithAttr(ctx, "meterSlug", meterSlug)
	ctx = contextx.WithAttr(ctx, "params", params) // TODO: we should probable NOT add this to the context

	subject, ok := authenticator.GetAuthenticatedSubject(ctx)
	if !ok {
		err := fmt.Errorf("not authenticated")
		models.NewStatusProblem(ctx, err, http.StatusUnauthorized).Respond(w)
		return
	}

	a.QueryMeter(w, r, meterSlug, api.QueryMeterParams{
		From:           params.From,
		To:             params.To,
		FilterGroupBy:  params.FilterGroupBy,
		WindowSize:     params.WindowSize,
		WindowTimeZone: params.WindowTimeZone,
		Subject:        &[]string{subject},
		GroupBy:        params.GroupBy,
	})
}
