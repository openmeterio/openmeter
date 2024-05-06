package router

import (
	"net/http"
	"time"

	"github.com/go-chi/render"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Get credit balance, GET /api/v1/ledgers/{creditSubjectId}/history
func (a *Router) GetCreditHistory(w http.ResponseWriter, r *http.Request, creditSubjectId string, params api.GetCreditHistoryParams) {
	ctx := contextx.WithAttr(r.Context(), "operation", "getCreditLedger")
	namespace := a.config.NamespaceManager.GetDefaultNamespace()

	// Get Ledger
	ledgerEntries, err := a.config.CreditConnector.GetHistory(
		ctx, namespace, creditSubjectId,
		params.From,
		defaultx.WithDefault(params.To, time.Now()),
		defaultx.WithDefault(params.Limit, api.DefaultCreditsQueryLimit),
	)
	if err != nil {
		a.config.ErrorHandler.HandleContext(ctx, err)
		models.NewStatusProblem(ctx, err, http.StatusInternalServerError).Respond(w, r)
		return
	}

	_ = render.Render(w, r, ledgerEntries)
}
