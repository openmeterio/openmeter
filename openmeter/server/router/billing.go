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
	if !a.config.BillingEnabled {
		unimplemented.ListBillingProfileCustomerOverrides(w, r, params)
		return
	}

	a.billingHandler.ListCustomerOverrides().With(params).ServeHTTP(w, r)
}

// Delete a customer override
// (DELETE /api/v1/billing/customer/{customerId})
func (a *Router) DeleteBillingProfileCustomerOverride(w http.ResponseWriter, r *http.Request, customerId string) {
	if !a.config.BillingEnabled {
		unimplemented.DeleteBillingProfileCustomerOverride(w, r, customerId)
		return
	}

	a.billingHandler.DeleteCustomerOverride().With(httpdriver.DeleteCustomerOverrideParams{
		CustomerId: customerId,
	}).ServeHTTP(w, r)
}

// Get a customer override
// (GET /api/v1/billing/customer/{customerId})
func (a *Router) GetBillingProfileCustomerOverride(w http.ResponseWriter, r *http.Request, customerId string, params api.GetBillingProfileCustomerOverrideParams) {
	if !a.config.BillingEnabled {
		unimplemented.GetBillingProfileCustomerOverride(w, r, customerId, params)
		return
	}

	a.billingHandler.GetCustomerOverride().With(httpdriver.GetCustomerOverrideParams{
		CustomerId: customerId,
		Expand:     params.Expand,
	}).ServeHTTP(w, r)
}

// Create/update a customer override
// (POST /api/v1/billing/customer/{customerId})
func (a *Router) UpsertBillingProfileCustomerOverride(w http.ResponseWriter, r *http.Request, customerId string) {
	if !a.config.BillingEnabled {
		unimplemented.UpsertBillingProfileCustomerOverride(w, r, customerId)
		return
	}

	a.billingHandler.UpsertCustomerOverride().With(httpdriver.UpsertCustomerOverrideParams{
		CustomerId: customerId,
	}).ServeHTTP(w, r)
}

// List invoices
// (GET /api/v1/billing/invoices)
func (a *Router) ListInvoices(w http.ResponseWriter, r *http.Request, params api.ListInvoicesParams) {
	if !a.config.BillingEnabled {
		unimplemented.ListInvoices(w, r, params)
		return
	}

	a.billingHandler.ListInvoices().With(params).ServeHTTP(w, r)
}

// Invoice a customer based on the pending line items
// (POST /api/v1/billing/invoices/{customerId}/invoice)
func (a *Router) InvoicePendingLinesAction(w http.ResponseWriter, r *http.Request) {
	if !a.config.BillingEnabled {
		unimplemented.InvoicePendingLinesAction(w, r)
		return
	}

	a.billingHandler.InvoicePendingLinesAction().ServeHTTP(w, r)
}

// Delete an invoice
// (DELETE /api/v1/billing/invoices/{customerId}/invoices/{invoiceId})
func (a *Router) DeleteInvoice(w http.ResponseWriter, r *http.Request, invoiceId string) {
	if !a.config.BillingEnabled {
		unimplemented.DeleteInvoice(w, r, invoiceId)
		return
	}

	a.billingHandler.DeleteInvoice().With(httpdriver.DeleteInvoiceParams{
		InvoiceID: invoiceId,
	}).ServeHTTP(w, r)
}

// Get an invoice
// (GET /api/v1/billing/invoices/{invoiceId})
func (a *Router) GetInvoice(w http.ResponseWriter, r *http.Request, invoiceId string, params api.GetInvoiceParams) {
	if !a.config.BillingEnabled {
		unimplemented.GetInvoice(w, r, invoiceId, params)
		return
	}

	a.billingHandler.GetInvoice().With(httpdriver.GetInvoiceParams{
		InvoiceID:           invoiceId,
		Expand:              lo.FromPtr(params.Expand),
		IncludeDeletedLines: lo.FromPtr(params.IncludeDeletedLines),
	}).ServeHTTP(w, r)
}

// Advance the invoice's state to the next status
// (POST /api/v1/billing/invoices/{invoiceId}/advance)
func (a *Router) AdvanceInvoiceAction(w http.ResponseWriter, r *http.Request, invoiceId string) {
	if !a.config.BillingEnabled {
		unimplemented.AdvanceInvoiceAction(w, r, invoiceId)
		return
	}

	a.billingHandler.ProgressInvoice(httpdriver.InvoiceProgressActionAdvance).
		With(httpdriver.ProgressInvoiceParams{
			InvoiceID: invoiceId,
		}).ServeHTTP(w, r)
}

// Send the invoice to the customer
// (POST /api/v1/billing/invoices/{invoiceId}/approve)
func (a *Router) ApproveInvoiceAction(w http.ResponseWriter, r *http.Request, invoiceId string) {
	if !a.config.BillingEnabled {
		unimplemented.ApproveInvoiceAction(w, r, invoiceId)
		return
	}

	a.billingHandler.ProgressInvoice(httpdriver.InvoiceProgressActionApprove).
		With(httpdriver.ProgressInvoiceParams{
			InvoiceID: invoiceId,
		}).ServeHTTP(w, r)
}

// Simulate an invoice for a customer
// (POST /api/v1/billing/customers/{customerId}/invoices/simulate)
func (a *Router) SimulateInvoice(w http.ResponseWriter, r *http.Request, customerId string) {
	if !a.config.BillingEnabled {
		unimplemented.SimulateInvoice(w, r, customerId)
		return
	}

	a.billingHandler.SimulateInvoice().With(httpdriver.SimulateInvoiceParams{
		CustomerID: customerId,
	}).ServeHTTP(w, r)
}

// Update an invoice
// (PUT /api/v1/billing/invoices/{invoiceId})
func (a *Router) UpdateInvoice(w http.ResponseWriter, r *http.Request, invoiceId string) {
	if !a.config.BillingEnabled {
		unimplemented.UpdateInvoice(w, r, invoiceId)
		return
	}

	a.billingHandler.UpdateInvoice().With(httpdriver.UpdateInvoiceParams{
		InvoiceID: invoiceId,
	}).ServeHTTP(w, r)
}

// Retry a failed synchronization step of the invoice
// (POST /api/v1/billing/invoices/{invoiceId}/retry)
func (a *Router) RetryInvoiceAction(w http.ResponseWriter, r *http.Request, invoiceId string) {
	if !a.config.BillingEnabled {
		unimplemented.RetryInvoiceAction(w, r, invoiceId)
		return
	}

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
// (POST /api/v1/billing/invoices/lines)
func (a *Router) CreatePendingInvoiceLine(w http.ResponseWriter, r *http.Request) {
	if !a.config.BillingEnabled {
		unimplemented.CreatePendingInvoiceLine(w, r)
		return
	}

	a.billingHandler.CreatePendingLine().ServeHTTP(w, r)
}

// (GET /api/v1/billing/profile)
func (a *Router) ListBillingProfiles(w http.ResponseWriter, r *http.Request, params api.ListBillingProfilesParams) {
	if !a.config.BillingEnabled {
		unimplemented.ListBillingProfiles(w, r, params)
		return
	}

	a.billingHandler.ListProfiles().With(params).ServeHTTP(w, r)
}

// Create a new billing profile
// (POST /api/v1/billing/profile)
func (a *Router) CreateBillingProfile(w http.ResponseWriter, r *http.Request) {
	if !a.config.BillingEnabled {
		unimplemented.CreateBillingProfile(w, r)
		return
	}

	a.billingHandler.CreateProfile().ServeHTTP(w, r)
}

// Archive a billing profile
// (DELETE /api/v1/billing/profile/{id})
func (a *Router) DeleteBillingProfile(w http.ResponseWriter, r *http.Request, id string) {
	if !a.config.BillingEnabled {
		unimplemented.DeleteBillingProfile(w, r, id)
		return
	}

	a.billingHandler.DeleteProfile().With(id).ServeHTTP(w, r)
}

// Get a billing profile by ID
// (GET /api/v1/billing/profile/{id})
func (a *Router) GetBillingProfile(w http.ResponseWriter, r *http.Request, id string, params api.GetBillingProfileParams) {
	if !a.config.BillingEnabled {
		unimplemented.GetBillingProfile(w, r, id, params)
		return
	}

	a.billingHandler.GetProfile().With(httpdriver.GetProfileParams{
		ID:     id,
		Expand: lo.FromPtrOr(params.Expand, nil),
	}).ServeHTTP(w, r)
}

// Update a billing profile
// (PUT /api/v1/billing/profile/{id})
func (a *Router) UpdateBillingProfile(w http.ResponseWriter, r *http.Request, id string) {
	if !a.config.BillingEnabled {
		unimplemented.UpdateBillingProfile(w, r, id)
		return
	}

	a.billingHandler.UpdateProfile().With(id).ServeHTTP(w, r)
}
