package router

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/server/authenticator"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CreatePortalToken creates a new portal token.
func (a *Router) CreatePortalToken(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("operation", "createPortalToken")

	if a.config.PortalTokenStrategy == nil {
		err := fmt.Errorf("not implemented: portal is not enabled")
		ErrorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented), w, r)
		return
	}

	// Parse request body
	body := &api.CreatePortalTokenJSONRequestBody{}
	if err := render.DecodeJSON(r.Body, body); err != nil {
		err := fmt.Errorf("decode json: %w", err)
		ErrorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusBadRequest), w, r)
		return
	}

	t, err := a.config.PortalTokenStrategy.Generate(body.Subject, body.AllowedMeterSlugs, body.ExpiresAt)
	if err != nil {
		err := fmt.Errorf("generate portal token: %w", err)
		ErrorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError), w, r)
		return
	}

	render.JSON(w, r, api.PortalToken{
		Id:                t.Id,
		Token:             t.Token,
		ExpiresAt:         t.ExpiresAt,
		Subject:           t.Subject,
		AllowedMeterSlugs: t.AllowedMeterSlugs,
	})
}

func (a *Router) ListPortalTokens(w http.ResponseWriter, r *http.Request, params api.ListPortalTokensParams) {
	logger := slog.With("operation", "listPortalTokens")
	err := fmt.Errorf("not implemented: portal token listing is an OpenMeter Cloud only feature")
	ErrorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented), w, r)
}

func (a *Router) InvalidatePortalTokens(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("operation", "invalidatePortalTokens")
	err := fmt.Errorf("not implemented: portal token invalidation is an OpenMeter Cloud only feature")
	ErrorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusNotImplemented), w, r)
}

func (a *Router) QueryPortalMeter(w http.ResponseWriter, r *http.Request, meterSlug string, params api.QueryPortalMeterParams) {
	logger := slog.With("operation", "queryPortalMeter", "meterSlug", meterSlug, "params", params)
	subject := authenticator.GetAuthenticatedSubject(r.Context())
	if subject == "" {
		err := fmt.Errorf("not authenticated")
		ErrorRespond(logger, models.NewStatusProblem(r.Context(), err, http.StatusUnauthorized), w, r)
		return
	}

	a.QueryMeter(w, r, meterSlug, api.QueryMeterParams{
		From:           params.From,
		To:             params.To,
		WindowSize:     params.WindowSize,
		WindowTimeZone: params.WindowTimeZone,
		Subject:        &[]string{subject},
		GroupBy:        params.GroupBy,
	})
}
