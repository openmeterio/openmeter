package server

import (
	"errors"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

func (s *Server) ListMeters(w http.ResponseWriter, r *http.Request, params api.ListMetersParams) {
	s.meterHandler.ListMeters().With(params).ServeHTTP(w, r)
}

func (s *Server) GetMeter(w http.ResponseWriter, r *http.Request, meterIdOrKey api.ULIDOrResourceKey) {
	apierrors.NewNotImplementedError(r.Context(), errors.New("not implemented")).HandleAPIError(w, r)
}

func (s *Server) CreateMeter(w http.ResponseWriter, r *http.Request) {
	apierrors.NewNotImplementedError(r.Context(), errors.New("not implemented")).HandleAPIError(w, r)
}
