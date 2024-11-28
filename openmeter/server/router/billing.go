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
	w.WriteHeader(http.StatusNotImplemented)
}

// Delete a customer override
// (DELETE /api/v1/billing/customer/{customerId})
func (a *Router) DeleteBillingProfileCustomerOverride(w http.ResponseWriter, r *http.Request, customerId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get a customer override
// (GET /api/v1/billing/customer/{customerId})
func (a *Router) GetBillingProfileCustomerOverride(w http.ResponseWriter, r *http.Request, customerId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create/update a customer override
// (POST /api/v1/billing/customer/{customerId})
func (a *Router) UpsertBillingProfileCustomerOverride(w http.ResponseWriter, r *http.Request, customerId string) {
	w.WriteHeader(http.StatusNotImplemented)
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

// List invoices
// (GET /api/v1/billing/invoices/{customerId})
func (a *Router) ListInvoicesByCustomer(w http.ResponseWriter, r *http.Request, customerId string, params api.ListInvoicesByCustomerParams) {
	if !a.config.BillingEnabled {
		unimplemented.ListInvoicesByCustomer(w, r, customerId, params)
		return
	}

	a.billingHandler.ListInvoices().With(a.billingHandler.ConvertListInvoicesByCustomerToListInvoices(customerId, params)).ServeHTTP(w, r)
}

// Invoice a customer based on the pending line items
// (POST /api/v1/billing/invoices/{customerId}/invoice)
func (a *Router) InvoicePendingLinesAction(w http.ResponseWriter, r *http.Request, customerId string) {
	if !a.config.BillingEnabled {
		unimplemented.InvoicePendingLinesAction(w, r, customerId)
		return
	}

	a.billingHandler.InvoicePendingLinesAction().With(customerId).ServeHTTP(w, r)
}

// Delete an invoice
// (DELETE /api/v1/billing/invoices/{customerId}/invoices/{invoiceId})
func (a *Router) DeleteInvoice(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string) {
	if !a.config.BillingEnabled {
		unimplemented.DeleteInvoice(w, r, customerId, invoiceId)
		return
	}

	a.billingHandler.DeleteInvoice().With(httpdriver.DeleteInvoiceParams{
		CustomerID: customerId,
		InvoiceID:  invoiceId,
	}).ServeHTTP(w, r)
}

// Get an invoice
// (GET /api/v1/billing/invoices/{customerId}/invoices/{invoiceId})
func (a *Router) GetInvoice(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string, params api.GetInvoiceParams) {
	if !a.config.BillingEnabled {
		unimplemented.GetInvoice(w, r, customerId, invoiceId, params)
		return
	}

	a.billingHandler.GetInvoice().With(httpdriver.GetInvoiceParams{
		CustomerID:          customerId,
		InvoiceID:           invoiceId,
		Expand:              params.Expand,
		IncludeDeletedLines: params.IncludeDeletedLines,
	}).ServeHTTP(w, r)
}

// Advance the invoice's state to the next status
// (POST /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/advance)
func (a *Router) AdvanceInvoiceAction(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string) {
	if !a.config.BillingEnabled {
		unimplemented.AdvanceInvoiceAction(w, r, customerId, invoiceId)
		return
	}

	a.billingHandler.ProgressInvoice(httpdriver.InvoiceProgressActionAdvance).
		With(httpdriver.ProgressInvoiceParams{
			CustomerID: customerId,
			InvoiceID:  invoiceId,
		}).ServeHTTP(w, r)
}

// Send the invoice to the customer
// (POST /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/approve)
func (a *Router) ApproveInvoiceAction(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string) {
	if !a.config.BillingEnabled {
		unimplemented.ApproveInvoiceAction(w, r, customerId, invoiceId)
		return
	}

	a.billingHandler.ProgressInvoice(httpdriver.InvoiceProgressActionApprove).
		With(httpdriver.ProgressInvoiceParams{
			CustomerID: customerId,
			InvoiceID:  invoiceId,
		}).ServeHTTP(w, r)
}

// Delete an invoice line
// (DELETE /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/lines/{lineId})
func (a *Router) DeleteInvoiceLine(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string, lineId string) {
	if !a.config.BillingEnabled {
		unimplemented.DeleteInvoiceLine(w, r, customerId, invoiceId, lineId)
		return
	}

	a.billingHandler.DeleteLine().With(httpdriver.DeleteLineParams{
		CustomerID: customerId,
		InvoiceID:  invoiceId,
		LineID:     lineId,
	}).ServeHTTP(w, r)
}

// Get an invoice line
// (GET /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/lines/{lineId})
func (a *Router) GetInvoiceLine(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string, lineId string) {
	if !a.config.BillingEnabled {
		unimplemented.GetInvoiceLine(w, r, customerId, invoiceId, lineId)
		return
	}

	a.billingHandler.GetLine().With(httpdriver.GetLineParams{
		CustomerID: customerId,
		InvoiceID:  invoiceId,
		LineID:     lineId,
	}).ServeHTTP(w, r)
}

// Update an invoice line
// (PUT /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/lines/{lineId})
func (a *Router) UpdateInvoiceLine(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string, lineId string) {
	if !a.config.BillingEnabled {
		unimplemented.UpdateInvoiceLine(w, r, customerId, invoiceId, lineId)
		return
	}

	a.billingHandler.UpdateLine().With(httpdriver.UpdateLineParams{
		CustomerID: customerId,
		InvoiceID:  invoiceId,
		LineID:     lineId,
	}).ServeHTTP(w, r)
}

// Retry a failed synchronization step of the invoice
// (POST /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/retry)
func (a *Router) RetryInvoiceAction(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string) {
	if !a.config.BillingEnabled {
		unimplemented.RetryInvoiceAction(w, r, customerId, invoiceId)
		return
	}

	a.billingHandler.ProgressInvoice(httpdriver.InvoiceProgressActionRetry).
		With(httpdriver.ProgressInvoiceParams{
			CustomerID: customerId,
			InvoiceID:  invoiceId,
		}).ServeHTTP(w, r)
}

// Recalculate an invoice's tax amounts
// (POST /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/tax/recalculate)
func (a *Router) RecalculateInvoiceTaxAction(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Void an invoice
// (POST /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/void)
func (a *Router) VoidInvoiceAction(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create a new  item
// (POST /api/v1/billing/invoices/{customerId}/items)
func (a *Router) CreateInvoiceLineByCustomer(w http.ResponseWriter, r *http.Request, customerId string) {
	if !a.config.BillingEnabled {
		unimplemented.CreateInvoiceLineByCustomer(w, r, customerId)
		return
	}

	a.billingHandler.CreateLineByCustomer().With(customerId).ServeHTTP(w, r)
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
