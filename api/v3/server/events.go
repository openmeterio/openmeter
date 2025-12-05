package server

import (
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

func (s *Server) IngestMeteringEvents(w http.ResponseWriter, r *http.Request) {
	apierrors.NewNotImplementedError(r.Context(), errors.New("not implemented")).HandleAPIError(w, r)
}
