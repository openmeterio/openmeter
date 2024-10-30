package router

import (
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
)

// List customer overrides
// (GET /api/v1/billing/customer)
func (a *Router) BillingListCustomerOverrides(w http.ResponseWriter, r *http.Request, params api.BillingListCustomerOverridesParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Delete a customer override
// (DELETE /api/v1/billing/customer/{customerIdOrKey})
func (a *Router) BillingDeleteCustomerOverride(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get a customer override
// (GET /api/v1/billing/customer/{customerIdOrKey})
func (a *Router) BillingGetCustomerOverrideById(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create/update a customer override
// (POST /api/v1/billing/customer/{customerIdOrKey})
func (a *Router) BillingUpsertCustomerOverride(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// List invoices
// (GET /api/v1/billing/invoices)
func (a *Router) BillingListInvoices(w http.ResponseWriter, r *http.Request, params api.BillingListInvoicesParams) {
	if !a.config.BillingEnabled {
		unimplemented.BillingListInvoices(w, r, params)
		return
	}

	a.billingHandler.ListInvoices().With(params).ServeHTTP(w, r)
}

// List invoices
// (GET /api/v1/billing/invoices/{customerIdOrKey})
func (a *Router) BillingListInvoicesByCustomer(w http.ResponseWriter, r *http.Request, customerIdOrKey string, params api.BillingListInvoicesByCustomerParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create an invoice
// (POST /api/v1/billing/invoices/{customerIdOrKey})
func (a *Router) BillingCreateInvoice(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Delete an invoice
// (DELETE /api/v1/billing/invoices/{customerIdOrKey}/invoices/{invoiceId})
func (a *Router) BillingDeleteInvoiceByCustomerInvoiceId(w http.ResponseWriter, r *http.Request, customerIdOrKey string, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get an invoice
// (GET /api/v1/billing/invoices/{customerIdOrKey}/invoices/{invoiceId})
func (a *Router) BillingGetInvoiceByCustomerInvoiceId(w http.ResponseWriter, r *http.Request, customerIdOrKey string, invoiceId string, params api.BillingGetInvoiceByCustomerInvoiceIdParams) {
	// TODO: make sure we expand lines if there's no expand query param
	w.WriteHeader(http.StatusNotImplemented)
}

// Send the invoice to the customer
// (POST /api/v1/billing/invoices/{customerIdOrKey}/invoices/{invoiceId}/approve)
func (a *Router) BillingApproveInvoice(w http.ResponseWriter, r *http.Request, customerIdOrKey string, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Delete an invoice line
// (DELETE /api/v1/billing/invoices/{customerIdOrKey}/invoices/{invoiceId}/lines/{lineId})
func (a *Router) BillingDeleteInvoiceLine(w http.ResponseWriter, r *http.Request, customerIdOrKey string, invoiceId string, lineId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Update an invoice line
// (PUT /api/v1/billing/invoices/{customerIdOrKey}/invoices/{invoiceId}/lines/{lineId})
func (a *Router) BillingUpdateInvoiceLine(w http.ResponseWriter, r *http.Request, customerIdOrKey string, invoiceId string, lineId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Recalculate an invoice's tax amounts
// (POST /api/v1/billing/invoices/{customerIdOrKey}/invoices/{invoiceId}/tax/recalculate)
func (a *Router) BillingRecalculateInvoiceTax(w http.ResponseWriter, r *http.Request, customerIdOrKey string, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Void an invoice
// (POST /api/v1/billing/invoices/{customerIdOrKey}/invoices/{invoiceId}/void)
func (a *Router) BillingVoidInvoice(w http.ResponseWriter, r *http.Request, customerIdOrKey string, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Advance the invoice's state to the next status
// (POST /api/v1/billing/invoices/{customerIdOrKey}/invoices/{invoiceId}/advance)
func (a *Router) BillingInvoiceAdvance(w http.ResponseWriter, r *http.Request, customerIdOrKey string, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Retry a failed synchronization step of the invoice
// (POST /api/v1/billing/invoices/{customerIdOrKey}/invoices/{invoiceId}/retry)
func (a *Router) BillingInvoiceRetry(w http.ResponseWriter, r *http.Request, customerIdOrKey string, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create a new  item
// (POST /api/v1/billing/invoices/{customerIdOrKey}/items)
func (a *Router) BillingCreateLineByCustomer(w http.ResponseWriter, r *http.Request, customerIdOrKey string) {
	if !a.config.BillingEnabled {
		unimplemented.BillingCreateLineByCustomer(w, r, customerIdOrKey)
		return
	}

	a.billingHandler.CreateLineByCustomer().With(customerIdOrKey).ServeHTTP(w, r)
}

// (GET /api/v1/billing/profile)
func (a *Router) BillingListProfiles(w http.ResponseWriter, r *http.Request, params api.BillingListProfilesParams) {
	if !a.config.BillingEnabled {
		unimplemented.BillingListProfiles(w, r, params)
		return
	}

	a.billingHandler.ListProfiles().With(params).ServeHTTP(w, r)
}

// Create a new billing profile
// (POST /api/v1/billing/profile)
func (a *Router) BillingCreateProfile(w http.ResponseWriter, r *http.Request) {
	if !a.config.BillingEnabled {
		unimplemented.BillingCreateProfile(w, r)
		return
	}

	a.billingHandler.CreateProfile().ServeHTTP(w, r)
}

// Archive a billing profile
// (DELETE /api/v1/billing/profile/{id})
func (a *Router) BillingArchiveProfile(w http.ResponseWriter, r *http.Request, id string) {
	if !a.config.BillingEnabled {
		unimplemented.BillingArchiveProfile(w, r, id)
		return
	}

	a.billingHandler.ArchiveProfile().With(id).ServeHTTP(w, r)
}

// Get a billing profile by ID
// (GET /api/v1/billing/profile/{id})
func (a *Router) BillingGetProfile(w http.ResponseWriter, r *http.Request, id string, params api.BillingGetProfileParams) {
	if !a.config.BillingEnabled {
		unimplemented.BillingGetProfile(w, r, id, params)
		return
	}

	a.billingHandler.GetProfile().With(httpdriver.GetProfileParams{
		ID:     id,
		Expand: lo.FromPtrOr(params.Expand, nil),
	}).ServeHTTP(w, r)
}

// Update a billing profile
// (PUT /api/v1/billing/profile/{id})
func (a *Router) BillingUpdateProfile(w http.ResponseWriter, r *http.Request, id string) {
	if !a.config.BillingEnabled {
		unimplemented.BillingUpdateProfile(w, r, id)
		return
	}

	a.billingHandler.UpdateProfile().With(id).ServeHTTP(w, r)
}
