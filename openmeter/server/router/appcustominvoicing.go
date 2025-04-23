package router

import (
	"net/http"

	appcustominvoicinghttpdriver "github.com/openmeterio/openmeter/openmeter/app/custominvoicing/httpdriver"
)

// Submit draft syncronization results
// (POST /api/v1/apps/custom-invoicing/{invoiceId}/draft/syncronized)
func (a *Router) AppCustomInvoicingDraftSyncronized(w http.ResponseWriter, r *http.Request, invoiceId string) {
	a.appCustomInvocingHandler.DraftSyncronized().With(appcustominvoicinghttpdriver.DraftSyncronizedParams{
		InvoiceID: invoiceId,
	}).ServeHTTP(w, r)
}

// Submit issuing syncronization results
// (POST /api/v1/apps/custom-invoicing/{invoiceId}/issuing/syncronized)
func (a *Router) AppCustomInvoicingIssuingSyncronized(w http.ResponseWriter, r *http.Request, invoiceId string) {
	a.appCustomInvocingHandler.IssuingSyncronized().With(appcustominvoicinghttpdriver.IssuingSyncronizedParams{
		InvoiceID: invoiceId,
	}).ServeHTTP(w, r)
}

// Update payment status
// (POST /api/v1/apps/custom-invoicing/{invoiceId}/payment/status)
func (a *Router) AppCustomInvoicingUpdatePaymentStatus(w http.ResponseWriter, r *http.Request, invoiceId string) {
	a.appCustomInvocingHandler.UpdatePaymentStatus().With(appcustominvoicinghttpdriver.UpdatePaymentStatusParams{
		InvoiceID: invoiceId,
	}).ServeHTTP(w, r)
}
