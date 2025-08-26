package router

import (
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
)

// List customer overrides
// (GET /api/v1/billing/customer)
func (a *Router) ListBillingProfileCustomerOverrides(w http.ResponseWriter, r *http.Request, params api.ListBillingProfileCustomerOverridesParams) {
	a.billingHandler.ListCustomerOverrides().With(params).ServeHTTP(w, r)
}

// Delete a customer override
// (DELETE /api/v1/billing/customer/{customerId})
func (a *Router) DeleteBillingProfileCustomerOverride(w http.ResponseWriter, r *http.Request, customerId string) {
	a.billingHandler.DeleteCustomerOverride().With(httpdriver.DeleteCustomerOverrideParams{
		CustomerId: customerId,
	}).ServeHTTP(w, r)
}

// Get a customer override
// (GET /api/v1/billing/customer/{customerId})
func (a *Router) GetBillingProfileCustomerOverride(w http.ResponseWriter, r *http.Request, customerId string, params api.GetBillingProfileCustomerOverrideParams) {
	a.billingHandler.GetCustomerOverride().With(httpdriver.GetCustomerOverrideParams{
		CustomerId: customerId,
		Expand:     params.Expand,
	}).ServeHTTP(w, r)
}

// Create/update a customer override
// (POST /api/v1/billing/customer/{customerId})
func (a *Router) UpsertBillingProfileCustomerOverride(w http.ResponseWriter, r *http.Request, customerId string) {
	a.billingHandler.UpsertCustomerOverride().With(httpdriver.UpsertCustomerOverrideParams{
		CustomerId: customerId,
	}).ServeHTTP(w, r)
}

// List invoices
// (GET /api/v1/billing/invoices)
func (a *Router) ListInvoices(w http.ResponseWriter, r *http.Request, params api.ListInvoicesParams) {
	a.billingHandler.ListInvoices().With(params).ServeHTTP(w, r)
}

// Invoice a customer based on the pending line items
// (POST /api/v1/billing/invoices/{customerId}/invoice)
func (a *Router) InvoicePendingLinesAction(w http.ResponseWriter, r *http.Request) {
	a.billingHandler.InvoicePendingLinesAction().ServeHTTP(w, r)
}

// Delete an invoice
// (DELETE /api/v1/billing/invoices/{customerId}/invoices/{invoiceId})
func (a *Router) DeleteInvoice(w http.ResponseWriter, r *http.Request, invoiceId string) {
	a.billingHandler.DeleteInvoice().With(httpdriver.DeleteInvoiceParams{
		InvoiceID: invoiceId,
	}).ServeHTTP(w, r)
}

// Get an invoice
// (GET /api/v1/billing/invoices/{invoiceId})
func (a *Router) GetInvoice(w http.ResponseWriter, r *http.Request, invoiceId string, params api.GetInvoiceParams) {
	a.billingHandler.GetInvoice().With(httpdriver.GetInvoiceParams{
		InvoiceID:           invoiceId,
		Expand:              lo.FromPtr(params.Expand),
		IncludeDeletedLines: lo.FromPtr(params.IncludeDeletedLines),
	}).ServeHTTP(w, r)
}

// Get an invoice line cost
// (GET /api/v1/billing/invoices/{invoiceId}/lines/{lineId}/cost)
func (a *Router) GetInvoiceLineCost(w http.ResponseWriter, r *http.Request, invoiceId string, lineId string, params api.GetInvoiceLineCostParams) {
	a.billingHandler.GetInvoiceLineCost().With(httpdriver.GetInvoiceLineCostParams{
		InvoiceID: invoiceId,
		LineID:    lineId,
		Params:    params,
	}).ServeHTTP(w, r)
}

// Snapshot quantities for usage based line items
// (POST /api/v1/billing/invoices/{invoiceId}/snapshot-quantities)
func (a *Router) SnapshotQuantitiesInvoiceAction(w http.ResponseWriter, r *http.Request, invoiceId string) {
	a.billingHandler.ProgressInvoice(httpdriver.InvoiceProgressActionSnapshotQuantities).
		With(httpdriver.ProgressInvoiceParams{
			InvoiceID: invoiceId,
		}).ServeHTTP(w, r)
}

// Advance the invoice's state to the next status
// (POST /api/v1/billing/invoices/{invoiceId}/advance)
func (a *Router) AdvanceInvoiceAction(w http.ResponseWriter, r *http.Request, invoiceId string) {
	a.billingHandler.ProgressInvoice(httpdriver.InvoiceProgressActionAdvance).
		With(httpdriver.ProgressInvoiceParams{
			InvoiceID: invoiceId,
		}).ServeHTTP(w, r)
}

// Send the invoice to the customer
// (POST /api/v1/billing/invoices/{invoiceId}/approve)
func (a *Router) ApproveInvoiceAction(w http.ResponseWriter, r *http.Request, invoiceId string) {
	a.billingHandler.ProgressInvoice(httpdriver.InvoiceProgressActionApprove).
		With(httpdriver.ProgressInvoiceParams{
			InvoiceID: invoiceId,
		}).ServeHTTP(w, r)
}

// Simulate an invoice for a customer
// (POST /api/v1/billing/customers/{customerId}/invoices/simulate)
func (a *Router) SimulateInvoice(w http.ResponseWriter, r *http.Request, customerId string) {
	a.billingHandler.SimulateInvoice().With(httpdriver.SimulateInvoiceParams{
		CustomerID: customerId,
	}).ServeHTTP(w, r)
}

// Update an invoice
// (PUT /api/v1/billing/invoices/{invoiceId})
func (a *Router) UpdateInvoice(w http.ResponseWriter, r *http.Request, invoiceId string) {
	a.billingHandler.UpdateInvoice().With(httpdriver.UpdateInvoiceParams{
		InvoiceID: invoiceId,
	}).ServeHTTP(w, r)
}

// Retry a failed synchronization step of the invoice
// (POST /api/v1/billing/invoices/{invoiceId}/retry)
func (a *Router) RetryInvoiceAction(w http.ResponseWriter, r *http.Request, invoiceId string) {
	a.billingHandler.ProgressInvoice(httpdriver.InvoiceProgressActionRetry).
		With(httpdriver.ProgressInvoiceParams{
			InvoiceID: invoiceId,
		}).ServeHTTP(w, r)
}

// Recalculate an invoice's tax amounts
// (POST /api/v1/billing/invoices/{invoiceId}/tax/recalculate)
func (a *Router) RecalculateInvoiceTaxAction(w http.ResponseWriter, r *http.Request, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Void an invoice
// (POST /api/v1/billing/invoices/{invoiceId}/void)
func (a *Router) VoidInvoiceAction(w http.ResponseWriter, r *http.Request, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create a new line item
// (POST /api/v1/billing/customers/{customerId}/pending-lines)
func (a *Router) CreatePendingInvoiceLine(w http.ResponseWriter, r *http.Request, customerId string) {
	a.billingHandler.CreatePendingLine().With(httpdriver.CreatePendingLineParams{
		CustomerID: customerId,
	}).ServeHTTP(w, r)
}

// (GET /api/v1/billing/profile)
func (a *Router) ListBillingProfiles(w http.ResponseWriter, r *http.Request, params api.ListBillingProfilesParams) {
	a.billingHandler.ListProfiles().With(params).ServeHTTP(w, r)
}

// Create a new billing profile
// (POST /api/v1/billing/profile)
func (a *Router) CreateBillingProfile(w http.ResponseWriter, r *http.Request) {
	a.billingHandler.CreateProfile().ServeHTTP(w, r)
}

// Archive a billing profile
// (DELETE /api/v1/billing/profile/{id})
func (a *Router) DeleteBillingProfile(w http.ResponseWriter, r *http.Request, id string) {
	a.billingHandler.DeleteProfile().With(id).ServeHTTP(w, r)
}

// Get a billing profile by ID
// (GET /api/v1/billing/profile/{id})
func (a *Router) GetBillingProfile(w http.ResponseWriter, r *http.Request, id string, params api.GetBillingProfileParams) {
	a.billingHandler.GetProfile().With(httpdriver.GetProfileParams{
		ID:     id,
		Expand: lo.FromPtr(params.Expand),
	}).ServeHTTP(w, r)
}

// Update a billing profile
// (PUT /api/v1/billing/profile/{id})
func (a *Router) UpdateBillingProfile(w http.ResponseWriter, r *http.Request, id string) {
	a.billingHandler.UpdateProfile().With(id).ServeHTTP(w, r)
}
