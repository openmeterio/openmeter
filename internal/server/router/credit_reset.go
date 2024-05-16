package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
)

// Resets the credit POST /api/v1/ledgers/{ledgerID}/reset
func (a *Router) ResetLedger(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID) {
	a.CreditHandlers.ResetLedger.With(ledgerID).ServeHTTP(w, r)
}
