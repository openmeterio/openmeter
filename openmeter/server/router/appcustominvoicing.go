package router

import (
	"net/http"

	appcustominvoicinghttpdriver "github.com/openmeterio/openmeter/openmeter/app/custominvoicing/httpdriver"
)

// Submit draft synchronization results
// (POST /api/v1/apps/custom-invoicing/{invoiceId}/draft/synchronized)
func (a *Router) AppCustomInvoicingDraftSyncronized(w http.ResponseWriter, r *http.Request, invoiceId string) {
	a.appCustomInvoicingHandler.DraftSyncronized().With(appcustominvoicinghttpdriver.DraftSyncronizedParams{
		InvoiceID: invoiceId,
	}).ServeHTTP(w, r)
}

// Submit issuing synchronization results
// (POST /api/v1/apps/custom-invoicing/{invoiceId}/issuing/synchronized)
func (a *Router) AppCustomInvoicingIssuingSyncronized(w http.ResponseWriter, r *http.Request, invoiceId string) {
	a.appCustomInvoicingHandler.IssuingSyncronized().With(appcustominvoicinghttpdriver.IssuingSyncronizedParams{
		InvoiceID: invoiceId,
	}).ServeHTTP(w, r)
}

// Update payment status
// (POST /api/v1/apps/custom-invoicing/{invoiceId}/payment/status)
func (a *Router) AppCustomInvoicingUpdatePaymentStatus(w http.ResponseWriter, r *http.Request, invoiceId string) {
	a.appCustomInvoicingHandler.UpdatePaymentStatus().With(appcustominvoicinghttpdriver.UpdatePaymentStatusParams{
		InvoiceID: invoiceId,
	}).ServeHTTP(w, r)
}
