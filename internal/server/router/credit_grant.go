package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// List credit grants, GET /api/v1/ledgers/grants
func (a *Router) ListCreditGrants(w http.ResponseWriter, r *http.Request, params api.ListCreditGrantsParams) {
	ctx := contextx.WithAttr(r.Context(), "operation", "listCreditGrants")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	ledgerIDs := []ulid.ULID{}
	if params.LedgerID != nil {
		ledgerIDs = append(ledgerIDs, *params.LedgerID)
	}

	// Get grants
	grants, err := a.config.CreditConnector.ListGrants(ctx, namespace, credit.ListGrantsParams{
		LedgerIDs:         ledgerIDs,
		FromHighWatermark: true,
		IncludeVoid:       true,
		Limit:             defaultx.WithDefault(params.Limit, api.DefaultCreditsQueryLimit),
	})
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	// Response
	list := slicesx.Map[credit.Grant, render.Renderer](grants, func(grant credit.Grant) render.Renderer {
		return grant
	})
	_ = render.RenderList(w, r, list)
}

// List credit grants, GET /api/v1/ledgers/{ledgerID}/grants
func (a *Router) ListCreditGrantsByLedger(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID, params api.ListCreditGrantsByLedgerParams) {
	ctx := contextx.WithAttr(r.Context(), "operation", "listCreditGrants")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Get grants
	grants, err := a.config.CreditConnector.ListGrants(ctx, namespace, credit.ListGrantsParams{
		LedgerIDs:         []ulid.ULID{ledgerID},
		FromHighWatermark: true,
		IncludeVoid:       true,
		Limit:             defaultx.WithDefault(params.Limit, api.DefaultCreditsQueryLimit),
	})
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	// Response
	list := slicesx.Map[credit.Grant, render.Renderer](grants, func(grant credit.Grant) render.Renderer {
		return grant
	})
	_ = render.RenderList(w, r, list)
}

// Create credit grant, POST /api/v1/ledgers/{creditSubjectId}/grants
func (a *Router) CreateCreditGrant(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID) {
	ctx := contextx.WithAttr(r.Context(), "operation", "createCreditGrant")
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
			if _, ok := err.(*credit.FeatureNotFoundError); ok {
				err := fmt.Errorf("feature not found: %s", *grant.FeatureID)
				models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
				return
			}

			a.config.ErrorHandler.HandleContext(ctx, err)
			models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
			return
		}
	}

	grant.LedgerID = ledgerID
	// Let's make sure we are not allowing the ID to be specified externally
	grant.ID = nil

	// Create credit
	g, err := a.config.CreditConnector.CreateGrant(ctx, namespace, *grant)
	if err != nil {
		if _, ok := err.(*credit.HighWatermarBeforeError); ok {
			a.config.ErrorHandler.HandleContext(ctx, err)
			models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
			return
		}

		if _, ok := err.(*credit.LockErrNotObtainedError); ok {
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

// Void credit grant, DELETE /api/v1/ledgers/{ledgerID}/grants/{creditGrantID}
func (a *Router) VoidCreditGrant(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID, creditGrantId api.CreditGrantID) {
	ctx := contextx.WithAttr(r.Context(), "operation", "voidCreditGrant")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Get grant
	grant, err := a.config.CreditConnector.GetGrant(ctx, namespace, creditGrantId)
	if err != nil {
		if _, ok := err.(*credit.GrantNotFoundError); ok {
			err := fmt.Errorf("grant not found: %s", creditGrantId)
			models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w, r)
			return
		}

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	if grant.LedgerID != ledgerID {
		a.config.ErrorHandler.HandleContext(ctx, &credit.GrantNotFoundError{GrantID: creditGrantId})
		models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w, r)
		return
	}

	if grant.Void {
		err := fmt.Errorf("grant already void: %s", creditGrantId)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
		return
	}

	// TODO: this and the next should happen in the same trns
	// Get balance to check if grant can be voided: not partially or fully used yet
	balance, err := a.config.CreditConnector.GetBalance(ctx, namespace, ledgerID, time.Now())
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
		if _, ok := err.(*credit.HighWatermarBeforeError); ok {
			a.config.ErrorHandler.HandleContext(ctx, err)
			models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
			return
		}

		if _, ok := err.(*credit.LockErrNotObtainedError); ok {
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

// Get credit, GET /api/v1/ledgers/{ledgerID}/grants/{creditGrantId}
func (a *Router) GetCreditGrant(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID, creditGrantId api.CreditGrantID) {
	ctx := contextx.WithAttr(r.Context(), "operation", "getCreditGrant")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Get grant
	grant, err := a.config.CreditConnector.GetGrant(ctx, namespace, creditGrantId)
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	if grant.LedgerID != ledgerID {
		a.config.ErrorHandler.HandleContext(ctx, &credit.GrantNotFoundError{GrantID: creditGrantId})
		models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w, r)
		return
	}

	_ = render.Render(w, r, grant)
}
