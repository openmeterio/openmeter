// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package router

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/server/authenticator"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CreatePortalToken creates a new portal token.
func (a *Router) CreatePortalToken(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "createPortalToken")

	if a.config.PortalTokenStrategy == nil {
		err := fmt.Errorf("not implemented: portal is not enabled")

		models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w)

		return
	}

	// Parse request body
	body := &api.CreatePortalTokenJSONRequestBody{}
	if err := render.DecodeJSON(r.Body, body); err != nil {
		err := fmt.Errorf("decode json: %w", err)

		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w)

		return
	}

	t, err := a.config.PortalTokenStrategy.Generate(body.Subject, body.AllowedMeterSlugs, body.ExpiresAt)
	if err != nil {
		err := fmt.Errorf("generate portal token: %w", err)

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w)

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
	ctx := contextx.WithAttr(r.Context(), "operation", "listPortalTokens")

	err := fmt.Errorf("not implemented: portal token listing is an OpenMeter Cloud only feature")

	models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w)
}

func (a *Router) InvalidatePortalTokens(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "invalidatePortalTokens")

	err := fmt.Errorf("not implemented: portal token invalidation is an OpenMeter Cloud only feature")

	models.NewStatusProblem(ctx, err, http.StatusNotImplemented).Respond(w)
}

func (a *Router) QueryPortalMeter(w http.ResponseWriter, r *http.Request, meterSlug string, params api.QueryPortalMeterParams) {
	ctx := contextx.WithAttr(r.Context(), "operation", "queryPortalMeter")
	ctx = contextx.WithAttr(ctx, "meterSlug", meterSlug)
	ctx = contextx.WithAttr(ctx, "params", params) // TODO: we should probable NOT add this to the context

	subject := authenticator.GetAuthenticatedSubject(ctx)
	if subject == "" {
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
