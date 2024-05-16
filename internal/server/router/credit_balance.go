package router

import (
	"net/http"

	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit/creditdriver"
)

// Get credit balance, GET /api/v1/ledgers/{ledgerID}/balance
func (a *Router) GetLedgerBalance(w http.ResponseWriter, r *http.Request, ledgerID ulid.ULID, params api.GetLedgerBalanceParams) {
	a.CreditHandlers.GetLedgerBalance.With(creditdriver.GetLedgerBalaceHandlerParams{
		LedgerID:    ledgerID,
		QueryParams: params,
	}).ServeHTTP(w, r)
}
