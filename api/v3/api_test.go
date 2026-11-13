package v3_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/models"
)

type meterServer struct {
	v3.Unimplemented
	params models.ListMetersParams
}

func (s *meterServer) ListMeters(w http.ResponseWriter, _ *http.Request, params models.ListMetersParams) {
	s.params = params
	w.WriteHeader(http.StatusNoContent)
}

func TestListMetersBindsModelFilter(t *testing.T) {
	server := &meterServer{}
	key := "requests"
	query := url.Values{"filter[key][eq]": {key}}
	req := httptest.NewRequest(http.MethodGet, "/openmeter/meters?"+query.Encode(), nil)
	resp := httptest.NewRecorder()

	v3.Handler(server).ServeHTTP(resp, req)

	require.Equal(t, http.StatusNoContent, resp.Code)
	require.Equal(t, &models.ListMetersParamsFilter{Key: &models.StringFieldFilter{Eq: &key}}, server.params.Filter)
}
