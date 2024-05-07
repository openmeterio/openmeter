package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Resets the credit POST /api/v1/ledgers/{ledgerID}/reset
func (a *Router) ResetCredit(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID) {
	ctx := contextx.WithAttr(r.Context(), "operation", "resetCredit")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Parse request body
	resetIn := &api.ResetCreditJSONRequestBody{}
	if err := render.DecodeJSON(r.Body, resetIn); err != nil {
		err := fmt.Errorf("decode json: %w", err)

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)

		return
	}

	// Check if reset date is in the future
	if resetIn.EffectiveAt.After(time.Now()) {
		err := fmt.Errorf("reset date cannot be in the future")
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
		return
	}

	resetIn.LedgerID = ledgerID

	// Reset credit
	reset, _, err := a.config.CreditConnector.Reset(ctx, namespace, *resetIn)
	if err != nil {
		if _, ok := err.(*credit.HighWatermarBeforeError); ok {
			models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
			return
		}

		if _, ok := err.(*credit.LockErrNotObtainedError); ok {
			err := fmt.Errorf("credit is currently locked, try again: %w", err)
			models.NewStatusProblem(ctx, err, http.StatusConflict).Respond(w, r)
			return
		}

		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	// Respond with reset
	render.Status(r, http.StatusCreated)
	_ = render.Render(w, r, reset)
}
