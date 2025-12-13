package server

import (
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
)

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
