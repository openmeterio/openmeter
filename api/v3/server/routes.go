package server

import (
	"errors"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

// Meters

func (s *Server) ListMeters(w http.ResponseWriter, r *http.Request, params api.ListMetersParams) {
	apierrors.NewNotImplementedError(r.Context(), errors.New("not implemented")).HandleAPIError(w, r)
}

func (s *Server) CreateMeter(w http.ResponseWriter, r *http.Request) {
	apierrors.NewNotImplementedError(r.Context(), errors.New("not implemented")).HandleAPIError(w, r)
}

func (s *Server) GetMeter(w http.ResponseWriter, r *http.Request, meterId api.ULID) {
	apierrors.NewNotImplementedError(r.Context(), errors.New("not implemented")).HandleAPIError(w, r)
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
	apierrors.NewNotImplementedError(r.Context(), errors.New("not implemented")).HandleAPIError(w, r)
}

func (s *Server) ListCustomers(w http.ResponseWriter, r *http.Request, params api.ListCustomersParams) {
	s.customersHandler.ListCustomers().With(params).ServeHTTP(w, r)
}

func (s *Server) UpsertCustomer(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	apierrors.NewNotImplementedError(r.Context(), errors.New("not implemented")).HandleAPIError(w, r)
}

func (s *Server) DeleteCustomer(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customersHandler.DeleteCustomer().With(customerId).ServeHTTP(w, r)
}

func (s *Server) CheckCustomerFeatureAccess(w http.ResponseWriter, r *http.Request, customerId api.ULID, featureKey api.ResourceKey) {
	apierrors.NewNotImplementedError(r.Context(), errors.New("not implemented")).HandleAPIError(w, r)
}

func (s *Server) CreateCustomerSubscription(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	apierrors.NewNotImplementedError(r.Context(), errors.New("not implemented")).HandleAPIError(w, r)
}

func (s *Server) ListCustomerSubscriptions(w http.ResponseWriter, r *http.Request, customerId api.ULID, params api.ListCustomerSubscriptionsParams) {
	apierrors.NewNotImplementedError(r.Context(), errors.New("not implemented")).HandleAPIError(w, r)
}
