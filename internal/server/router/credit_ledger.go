package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit/creditdriver"
)

// Get credit balance, GET /api/v1/ledgers/{ledgerID}/history
func (a *Router) GetLedgerHistory(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID, params api.GetLedgerHistoryParams) {
	a.CreditHandlers.GetLedgerHistory.With(creditdriver.GetLedgerHistoryRequest{
		GetLedgerHistoryParams: params,
		LedgerID:               ledgerID,
	}).ServeHTTP(w, r)
}

// CreateLedger POST /api/v1/ledgers
func (a *Router) CreateLedger(w http.ResponseWriter, r *http.Request) {
	a.CreditHandlers.CreateLedger.ServeHTTP(w, r)
}

// ListLedgers GET /api/v1/ledgers?subject=X&offset=Y&limit=Z
func (a *Router) ListLedgers(w http.ResponseWriter, r *http.Request, params api.ListLedgersParams) {
	a.CreditHandlers.ListLedgers.With(params).ServeHTTP(w, r)
}
