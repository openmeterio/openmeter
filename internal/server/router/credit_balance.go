package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Get credit balance, GET /api/v1/ledgers/{creditSubjectId}/balance
func (a *Router) GetCreditBalance(w http.ResponseWriter, r *http.Request, subject string, params api.GetCreditBalanceParams) {
	ctx := contextx.WithAttr(r.Context(), "operation", "getCreditBalance")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	cutline := time.Now()

	if params.Time != nil {
		cutline = *params.Time
	}

	// Get high watermark for credit
	hw, err := a.config.CreditConnector.GetHighWatermark(ctx, namespace, subject)
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	// Check if query time is after high watermark
	if cutline.Before(hw.Time) {
		err := fmt.Errorf("query time must be after high watermark: %s", hw.Time.Format(time.RFC3339))
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusBadRequest).Respond(w, r)
		return
	}

	// Get balance
	balance, err := a.config.CreditConnector.GetBalance(ctx, namespace, subject, cutline)
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	// Response
	_ = render.Render(w, r, balance)
}
