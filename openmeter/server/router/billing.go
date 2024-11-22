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
// (DELETE /api/v1/billing/customer/{customerId})
func (a *Router) BillingDeleteCustomerOverride(w http.ResponseWriter, r *http.Request, customerId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get a customer override
// (GET /api/v1/billing/customer/{customerId})
func (a *Router) BillingGetCustomerOverrideById(w http.ResponseWriter, r *http.Request, customerId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create/update a customer override
// (POST /api/v1/billing/customer/{customerId})
func (a *Router) BillingUpsertCustomerOverride(w http.ResponseWriter, r *http.Request, customerId string) {
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
// (GET /api/v1/billing/invoices/{customerId})
func (a *Router) BillingListInvoicesByCustomer(w http.ResponseWriter, r *http.Request, customerId string, params api.BillingListInvoicesByCustomerParams) {
	if !a.config.BillingEnabled {
		unimplemented.BillingListInvoicesByCustomer(w, r, customerId, params)
		return
	}

	a.billingHandler.ListInvoices().With(a.billingHandler.ConvertListInvoicesByCustomerToListInvoices(customerId, params)).ServeHTTP(w, r)
}

// Create an invoice
// (POST /api/v1/billing/invoices/{customerId})
func (a *Router) BillingCreateInvoice(w http.ResponseWriter, r *http.Request, customerId string) {
	if !a.config.BillingEnabled {
		unimplemented.BillingCreateInvoice(w, r, customerId)
		return
	}

	a.billingHandler.CreateInvoice().With(customerId).ServeHTTP(w, r)
}

// Delete an invoice
// (DELETE /api/v1/billing/invoices/{customerId}/invoices/{invoiceId})
func (a *Router) BillingDeleteInvoiceByCustomerInvoiceId(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get an invoice
// (GET /api/v1/billing/invoices/{customerId}/invoices/{invoiceId})
func (a *Router) BillingGetInvoiceByCustomerInvoiceId(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string, params api.BillingGetInvoiceByCustomerInvoiceIdParams) {
	if !a.config.BillingEnabled {
		unimplemented.BillingGetInvoiceByCustomerInvoiceId(w, r, customerId, invoiceId, params)
		return
	}

	a.billingHandler.GetInvoice().With(httpdriver.GetInvoiceParams{
		CustomerID: customerId,
		InvoiceID:  invoiceId,
		Expand:     params.Expand,
	}).ServeHTTP(w, r)
}

// Send the invoice to the customer
// (POST /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/approve)
func (a *Router) BillingApproveInvoice(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string) {
	if !a.config.BillingEnabled {
		unimplemented.BillingApproveInvoice(w, r, customerId, invoiceId)
		return
	}

	a.billingHandler.ProgressInvoice(httpdriver.InvoiceProgressActionApprove).
		With(httpdriver.ProgressInvoiceParams{
			CustomerID: customerId,
			InvoiceID:  invoiceId,
		}).ServeHTTP(w, r)
}

// Get an invoice line
// (GET /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/lines/{lineId})
func (a *Router) BillingGetInvoiceLine(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string, lineId string) {
	if !a.config.BillingEnabled {
		unimplemented.BillingGetInvoiceLine(w, r, customerId, invoiceId, lineId)
		return
	}

	a.billingHandler.GetLine().With(httpdriver.GetLineParams{
		CustomerID: customerId,
		InvoiceID:  invoiceId,
		LineID:     lineId,
	}).ServeHTTP(w, r)
}

// Delete an invoice line
// (DELETE /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/lines/{lineId})
func (a *Router) BillingDeleteInvoiceLine(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string, lineId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Update an invoice line
// (PUT /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/lines/{lineId})
func (a *Router) BillingUpdateInvoiceLine(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string, lineId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Recalculate an invoice's tax amounts
// (POST /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/tax/recalculate)
func (a *Router) BillingRecalculateInvoiceTax(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Void an invoice
// (POST /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/void)
func (a *Router) BillingVoidInvoice(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Advance the invoice's state to the next status
// (POST /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/advance)
func (a *Router) BillingInvoiceAdvance(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string) {
	if !a.config.BillingEnabled {
		unimplemented.BillingInvoiceAdvance(w, r, customerId, invoiceId)
		return
	}

	a.billingHandler.ProgressInvoice(httpdriver.InvoiceProgressActionAdvance).
		With(httpdriver.ProgressInvoiceParams{
			CustomerID: customerId,
			InvoiceID:  invoiceId,
		}).ServeHTTP(w, r)
}

// Retry a failed synchronization step of the invoice
// (POST /api/v1/billing/invoices/{customerId}/invoices/{invoiceId}/retry)
func (a *Router) BillingInvoiceRetry(w http.ResponseWriter, r *http.Request, customerId string, invoiceId string) {
	if !a.config.BillingEnabled {
		unimplemented.BillingInvoiceRetry(w, r, customerId, invoiceId)
		return
	}

	a.billingHandler.ProgressInvoice(httpdriver.InvoiceProgressActionRetry).
		With(httpdriver.ProgressInvoiceParams{
			CustomerID: customerId,
			InvoiceID:  invoiceId,
		}).ServeHTTP(w, r)
}

// Create a new  item
// (POST /api/v1/billing/invoices/{customerId}/items)
func (a *Router) BillingCreateLineByCustomer(w http.ResponseWriter, r *http.Request, customerId string) {
	if !a.config.BillingEnabled {
		unimplemented.BillingCreateLineByCustomer(w, r, customerId)
		return
	}

	a.billingHandler.CreateLineByCustomer().With(customerId).ServeHTTP(w, r)
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
