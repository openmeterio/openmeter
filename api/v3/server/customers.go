package server

import (
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
)

func (s *Server) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	s.customerHandler.CreateCustomer().ServeHTTP(w, r)
}

func (s *Server) GetCustomer(w http.ResponseWriter, r *http.Request, customerId api.ULID) {
	s.customerHandler.GetCustomer().With(customerId).ServeHTTP(w, r)
}

func (s *Server) ListCustomers(w http.ResponseWriter, r *http.Request, params api.ListCustomersParams) {
	s.customerHandler.ListCustomers().With(params).ServeHTTP(w, r)
}
