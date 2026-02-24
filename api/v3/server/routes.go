package server

import (
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
)

var unimplemented = api.Unimplemented{}

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

// Apps

func (s *Server) ListApps(w http.ResponseWriter, r *http.Request, params api.ListAppsParams) {
	s.appsHandler.ListApps().With(params).ServeHTTP(w, r)
}

func (s *Server) GetApp(w http.ResponseWriter, r *http.Request, appId api.ULID) {
	s.appsHandler.GetApp().With(appId).ServeHTTP(w, r)
}

// Billing Profiles

func (s *Server) ListBillingProfiles(w http.ResponseWriter, r *http.Request, params api.ListBillingProfilesParams) {
	s.billingProfilesHandler.ListBillingProfiles().With(params).ServeHTTP(w, r)
}

func (s *Server) CreateBillingProfile(w http.ResponseWriter, r *http.Request) {
	s.billingProfilesHandler.CreateBillingProfile().ServeHTTP(w, r)
}

func (s *Server) DeleteBillingProfile(w http.ResponseWriter, r *http.Request, id api.ULID) {
	s.billingProfilesHandler.DeleteBillingProfile().With(id).ServeHTTP(w, r)
}

func (s *Server) GetBillingProfile(w http.ResponseWriter, r *http.Request, id api.ULID) {
	s.billingProfilesHandler.GetBillingProfile().With(id).ServeHTTP(w, r)
}

func (s *Server) UpdateBillingProfile(w http.ResponseWriter, r *http.Request, id api.ULID) {
	s.billingProfilesHandler.UpdateBillingProfile().With(id).ServeHTTP(w, r)
}

// Customer Billing

func (s *Server) GetCustomerBilling(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersBillingHandler.GetCustomerBilling().With(customerId).ServeHTTP(w, r)
}

func (s *Server) UpdateCustomerBilling(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersBillingHandler.UpdateCustomerBilling().With(customerId).ServeHTTP(w, r)
}

func (s *Server) UpdateCustomerBillingAppData(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersBillingHandler.UpdateCustomerBillingAppData().With(customerId).ServeHTTP(w, r)
}

func (s *Server) CreateCustomerStripeCheckoutSession(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersBillingHandler.CreateCustomerStripeCheckoutSession().With(customerId).ServeHTTP(w, r)
}

func (s *Server) CreateCustomerStripePortalSession(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersBillingHandler.CreateCustomerStripePortalSession().With(customerId).ServeHTTP(w, r)
}

// Tax Codes

func (s *Server) ListTaxCodes(w http.ResponseWriter, r *http.Request, params api.ListTaxCodesParams) {
	unimplemented.ListTaxCodes(w, r, params)
}

func (s *Server) CreateTaxCode(w http.ResponseWriter, r *http.Request) {
	s.taxcodesHandler.CreateTaxCode().ServeHTTP(w, r)
}

func (s *Server) DeleteTaxCode(w http.ResponseWriter, r *http.Request, taxCodeId api.ULID) {
	unimplemented.DeleteTaxCode(w, r, taxCodeId)
}

func (s *Server) GetTaxCode(w http.ResponseWriter, r *http.Request, taxCodeId api.ULID) {
	unimplemented.GetTaxCode(w, r, taxCodeId)
}

func (s *Server) UpsertTaxCode(w http.ResponseWriter, r *http.Request, taxCodeId api.ULID) {
	unimplemented.UpsertTaxCode(w, r, taxCodeId)
}

// Currencies

func (s *Server) ListCurrencies(w http.ResponseWriter, r *http.Request, params api.ListCurrenciesParams) {
	unimplemented.ListCurrencies(w, r, params)
}

func (s *Server) CreateCustomCurrency(w http.ResponseWriter, r *http.Request) {
	unimplemented.CreateCustomCurrency(w, r)
}

func (s *Server) CreateCostBasis(w http.ResponseWriter, r *http.Request, currencyId api.ULID) {
	unimplemented.CreateCostBasis(w, r, currencyId)
}

func (s *Server) ListCostBases(w http.ResponseWriter, r *http.Request, currencyId api.ULID, params api.ListCostBasesParams) {
	unimplemented.ListCostBases(w, r, currencyId, params)
}
