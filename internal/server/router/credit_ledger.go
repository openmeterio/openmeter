package router

import (
	"net/http"
	"time"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Get credit balance, GET /api/v1/credit-ledger/{subject}
func (a *Router) GetCreditLedger(w http.ResponseWriter, r *http.Request, subject string, params api.GetCreditLedgerParams) {
	ctx := contextx.WithAttr(r.Context(), "operation", "GetCreditLedger")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()
	now := time.Now()
	to := now

	if params.To != nil {
		to = *params.To
	}

	// Get Ledger
	ledgerEntries, err := a.config.CreditConnector.GetHistory(ctx, namespace, subject, params.From, to)
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	_ = render.Render(w, r, ledgerEntries)
}
