package server

import (
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
)

// Apps

func (s *Server) ListAppCatalogItems(w http.ResponseWriter, r *http.Request, params api.ListAppCatalogItemsParams) {
	s.appsHandler.ListAppCatalogItems().With(params).ServeHTTP(w, r)
}

func (s *Server) GetAppCatalogItem(w http.ResponseWriter, r *http.Request, pType api.BillingAppType) {
	s.appsHandler.GetAppCatalogItem().With(pType).ServeHTTP(w, r)
}

func (s *Server) GetAppCatalogItemOauth2InstallUrl(w http.ResponseWriter, r *http.Request, pType api.BillingAppType) {
	s.appsHandler.GetAppCatalogItemOauth2InstallUrl().With(pType).ServeHTTP(w, r)
}

func (s *Server) SubmitCustomInvoicingDraftSynchronized(w http.ResponseWriter, r *http.Request, invoiceId api.ULID) {
	s.appsHandler.SubmitCustomInvoicingDraftSynchronized().With(invoiceId).ServeHTTP(w, r)
}

func (s *Server) SubmitCustomInvoicingIssuingSynchronized(w http.ResponseWriter, r *http.Request, invoiceId api.ULID) {
	s.appsHandler.SubmitCustomInvoicingIssuingSynchronized().With(invoiceId).ServeHTTP(w, r)
}

func (s *Server) UpdateCustomInvoicingPaymentStatus(w http.ResponseWriter, r *http.Request, invoiceId api.ULID) {
	s.appsHandler.UpdateCustomInvoicingPaymentStatus().With(invoiceId).ServeHTTP(w, r)
}

func (s *Server) CreateStripeCheckoutSession(w http.ResponseWriter, r *http.Request) {
	s.appsHandler.CreateStripeCheckoutSession().ServeHTTP(w, r)
}

func (s *Server) HandleStripeWebhook(w http.ResponseWriter, r *http.Request, appId api.ULID) {
	s.appsHandler.HandleStripeWebhook().With(appId).ServeHTTP(w, r)
}

// Meters
func (s *Server) CreateMeter(w http.ResponseWriter, r *http.Request) {
	s.metersHandler.CreateMeter().ServeHTTP(w, r)
}

func (s *Server) GetMeter(w http.ResponseWriter, r *http.Request, meterId api.ULID) {
	s.metersHandler.GetMeter().With(meterId).ServeHTTP(w, r)
}

func (s *Server) ListMeters(w http.ResponseWriter, r *http.Request, params api.ListMetersParams) {
	s.metersHandler.ListMeters().With(params).ServeHTTP(w, r)
}

func (s *Server) DeleteMeter(w http.ResponseWriter, r *http.Request, meterId api.ULID) {
	s.metersHandler.DeleteMeter().With(meterId).ServeHTTP(w, r)
}

// Events

func (s *Server) IngestMeteringEvents(w http.ResponseWriter, r *http.Request) {
	s.eventsHandler.IngestEvents().ServeHTTP(w, r)
}

// Customers

func (s *Server) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	s.customersHandler.CreateCustomer().ServeHTTP(w, r)
}

func (s *Server) GetCustomer(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersHandler.GetCustomer().With(customerId).ServeHTTP(w, r)
}

func (s *Server) ListCustomers(w http.ResponseWriter, r *http.Request, params api.ListCustomersParams) {
	s.customersHandler.ListCustomers().With(params).ServeHTTP(w, r)
}

func (s *Server) UpsertCustomer(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersHandler.UpsertCustomer().With(customerId).ServeHTTP(w, r)
}

func (s *Server) DeleteCustomer(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersHandler.DeleteCustomer().With(customerId).ServeHTTP(w, r)
}

// Customers Entitlement Access

func (s *Server) ListCustomerEntitlementAccess(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersEntitlementHandler.ListCustomerEntitlementAccess().With(customerId).ServeHTTP(w, r)
}

// Subscriptions

func (s *Server) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	s.subscriptionsHandler.CreateSubscription().ServeHTTP(w, r)
}

func (s *Server) ListSubscriptions(w http.ResponseWriter, r *http.Request, params api.ListSubscriptionsParams) {
	s.subscriptionsHandler.ListSubscriptions().With(params).ServeHTTP(w, r)
}

func (s *Server) GetSubscription(w http.ResponseWriter, r *http.Request, subscriptionId api.ULID) {
	s.subscriptionsHandler.GetSubscription().With(subscriptionId).ServeHTTP(w, r)
}

func (s *Server) CancelSubscription(w http.ResponseWriter, r *http.Request, subscriptionId api.ULID) {
	s.subscriptionsHandler.CancelSubscription().With(subscriptionId).ServeHTTP(w, r)
}

func (s *Server) UnscheduleCancelation(w http.ResponseWriter, r *http.Request, subscriptionId api.ULID) {
	s.subscriptionsHandler.UnscheduleCancelation().With(subscriptionId).ServeHTTP(w, r)
}

func (s *Server) ChangeSubscription(w http.ResponseWriter, r *http.Request, subscriptionId api.ULID) {
	s.subscriptionsHandler.ChangeSubscription().With(subscriptionId).ServeHTTP(w, r)
}
