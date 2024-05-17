package router

import (
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit/creditdriver"
)

// List credit grants, GET /api/v1/ledgers/grants
func (a *Router) ListLedgerGrants(w http.ResponseWriter, r *http.Request, params api.ListLedgerGrantsParams) {
	a.CreditHandlers.ListLedgerGrants.With(params).ServeHTTP(w, r)
}

// List credit grants, GET /api/v1/ledgers/{ledgerID}/grants
func (a *Router) ListLedgerGrantsByLedger(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID, params api.ListLedgerGrantsByLedgerParams) {
	a.CreditHandlers.ListLedgerGrantsByLedger.With(creditdriver.ListLedgerGrantsByLedgerParams{
		LedgerID: ledgerID,
		Params:   params,
	}).ServeHTTP(w, r)
}

// Create credit grant, POST /api/v1/ledgers/{creditSubjectId}/grants
func (a *Router) CreateLedgerGrant(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID) {
	a.CreditHandlers.CreateLedgerGrant.With(ledgerID).ServeHTTP(w, r)
}

// Void credit grant, DELETE /api/v1/ledgers/{ledgerID}/grants/{creditGrantID}
func (a *Router) VoidLedgerGrant(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID, creditGrantId api.LedgerGrantID) {
	a.CreditHandlers.VoidLedgerGrant.With(creditdriver.GrantPathParams{
		LedgerID: ledgerID,
		GrantID:  creditGrantId,
	}).ServeHTTP(w, r)
}

// Get credit, GET /api/v1/ledgers/{ledgerID}/grants/{creditGrantId}
func (a *Router) GetLedgerGrant(w http.ResponseWriter, r *http.Request, ledgerID api.LedgerID, creditGrantId api.LedgerGrantID) {
	a.CreditHandlers.GetLedgerGrant.With(creditdriver.GrantPathParams{
		LedgerID: ledgerID,
		GrantID:  creditGrantId,
	}).ServeHTTP(w, r)

}
