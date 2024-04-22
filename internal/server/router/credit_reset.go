package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	credit_model "github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Resets the credit POST /api/v1/credit-resets
func (a *Router) ResetCredit(w http.ResponseWriter, r *http.Request) {
	ctx := contextx.WithAttr(r.Context(), "operation", "ResetCredit")
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
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
		return
	}

	// Check if reset is after high watermark
	hw, err := a.config.CreditConnector.GetHighWatermark(ctx, namespace, resetIn.Subject)
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}
	if !resetIn.EffectiveAt.After(hw.Time) {
		err := fmt.Errorf("new resets must be after last reset: %s", hw.Time.Format(time.RFC3339))
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
		return
	}

	// Reset credit
	reset, _, err := a.config.CreditConnector.Reset(ctx, namespace, *resetIn)
	if err != nil {
		if _, ok := err.(*credit_model.LockErrNotObtained); ok {
			err := fmt.Errorf("credit is currently locked, try again: %w", err)
			a.config.ErrorHandler.HandleContext(ctx, err)
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
