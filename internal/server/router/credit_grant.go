package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	credit_connector "github.com/openmeterio/openmeter/internal/credit"
	credit_model "github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// List credit grants, GET /api/v1/credit-grants
func (a *Router) ListCreditGrants(w http.ResponseWriter, r *http.Request, params api.ListCreditGrantsParams) {
	ctx := contextx.WithAttr(r.Context(), "operation", "ListCreditGrants")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	var subjects []string
	if params.Subject != nil {
		subjects = *params.Subject
	}

	// Get grants
	grants, err := a.config.CreditConnector.ListGrants(ctx, namespace, credit_connector.ListGrantsParams{
		Subjects:          subjects,
		FromHighWatermark: true,
		IncludeVoid:       true,
	})
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	// Response
	list := slicesx.Map[credit_model.Grant, render.Renderer](grants, func(grant credit_model.Grant) render.Renderer {
		return &grant
	})
	_ = render.RenderList(w, r, list)
}

// Create credit grant, POST /api/v1/credit-grants
func (a *Router) CreateCreditGrant(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "CreateCreditGrant")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Parse request body
	grant := &api.CreateCreditGrantJSONRequestBody{}
	if err := render.DecodeJSON(r.Body, grant); err != nil {
		err := fmt.Errorf("decode json: %w", err)

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)

		return
	}

	if grant.Priority == 0 {
		grant.Priority = 1
	}

	// Check if feature exists
	if grant.FeatureID != nil {
		_, err := a.config.CreditConnector.GetFeature(ctx, namespace, *grant.FeatureID)
		if err != nil {
			if _, ok := err.(*credit_model.FeatureNotFoundError); ok {
				err := fmt.Errorf("feature not found: %s", *grant.FeatureID)
				models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
				return
			}

			a.config.ErrorHandler.HandleContext(ctx, err)
			models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
			return
		}
	}

	// Create credit
	g, err := a.config.CreditConnector.CreateGrant(ctx, namespace, *grant)
	if err != nil {
		if _, ok := err.(*credit_model.HighWatermarBeforeError); ok {
			a.config.ErrorHandler.HandleContext(ctx, err)
			models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
			return
		}

		if _, ok := err.(*credit_model.LockErrNotObtainedError); ok {
			err := fmt.Errorf("credit is currently locked, try again: %w", err)
			a.config.ErrorHandler.HandleContext(ctx, err)
			models.NewStatusProblem(ctx, err, http.StatusConflict).Respond(w, r)
			return
		}

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	render.Status(r, http.StatusCreated)
	_ = render.Render(w, r, g)
}

// Void credit grant, DELETE /api/v1/credit-grants/{creditGrantId}
func (a *Router) VoidCreditGrant(w http.ResponseWriter, r *http.Request, creditGrantId api.CreditGrantId) {
	ctx := contextx.WithAttr(r.Context(), "operation", "VoidCreditGrant")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Get grant
	grant, err := a.config.CreditConnector.GetGrant(ctx, namespace, creditGrantId)
	if err != nil {
		if _, ok := err.(*credit_model.GrantNotFoundError); ok {
			err := fmt.Errorf("grant not found: %s", creditGrantId)
			models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w, r)
			return
		}

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	if grant.Void {
		err := fmt.Errorf("grant already void: %s", creditGrantId)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
		return
	}

	// Get balance to check if grant can be voided: not partially or fully used yet
	balance, err := a.config.CreditConnector.GetBalance(ctx, namespace, grant.Subject, time.Now())
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}
	for _, entry := range balance.GrantBalances {
		if *entry.Grant.ID == *grant.ID {
			if entry.Balance != grant.Amount {
				err := fmt.Errorf("grant has been used, cannot void: %s", creditGrantId)
				models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
				return
			}
			break
		}

	}

	// Void grant
	_, err = a.config.CreditConnector.VoidGrant(ctx, namespace, grant)
	if err != nil {
		if _, ok := err.(*credit_model.HighWatermarBeforeError); ok {
			a.config.ErrorHandler.HandleContext(ctx, err)
			models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
			return
		}

		if _, ok := err.(*credit_model.LockErrNotObtainedError); ok {
			err := fmt.Errorf("credit is currently locked, try again: %w", err)
			a.config.ErrorHandler.HandleContext(ctx, err)
			models.NewStatusProblem(ctx, err, http.StatusConflict).Respond(w, r)
			return
		}

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	render.Status(r, http.StatusNoContent)
}

// Get credit, GET /api/v1/credit-grants/{creditGrantId}
func (a *Router) GetCreditGrant(w http.ResponseWriter, r *http.Request, creditGrantId api.CreditGrantId) {
	ctx := contextx.WithAttr(r.Context(), "operation", "GetCreditGrant")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Get grant
	grant, err := a.config.CreditConnector.GetGrant(ctx, namespace, creditGrantId)
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	_ = render.Render(w, r, grant)
}
