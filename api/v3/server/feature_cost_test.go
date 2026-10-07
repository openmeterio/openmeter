package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/handlers/featurecost"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

func TestQueryFeatureCostContentNegotiation(t *testing.T) {
	s := Server{featureCostHandler: &featureCostRouteHandler{t: t}}
	for _, tt := range []struct {
		accept string
		want   string
	}{
		{want: "application/json"},
		{accept: "application/json", want: "application/json"},
		{accept: "text/csv", want: "text/csv"},
		{accept: "text/csv; charset=utf-8", want: "text/csv"},
	} {
		t.Run(tt.accept, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("Accept", tt.accept)
			res := httptest.NewRecorder()
			s.QueryFeatureCost(res, req, "feature-1")
			require.Equal(t, http.StatusOK, res.Code)
			require.Equal(t, tt.want, res.Header().Get("Content-Type"))
		})
	}
}

type featureCostRouteHandler struct {
	featurecost.Handler
	t *testing.T
}

func (h *featureCostRouteHandler) QueryFeatureCost() featurecost.QueryFeatureCostHandler {
	return httptransport.NewHandlerWithArgs(
		func(_ context.Context, _ *http.Request, id string) (featurecost.QueryFeatureCostRequest, error) {
			require.Equal(h.t, "feature-1", id)
			return featurecost.QueryFeatureCostRequest{}, nil
		},
		func(context.Context, featurecost.QueryFeatureCostRequest) (featurecost.QueryFeatureCostResponse, error) {
			return featurecost.QueryFeatureCostResponse{}, nil
		},
		commonhttp.JSONResponseEncoder[featurecost.QueryFeatureCostResponse],
	)
}

func (h *featureCostRouteHandler) QueryFeatureCostCSV() featurecost.QueryFeatureCostCSVHandler {
	return httptransport.NewHandlerWithArgs(
		func(_ context.Context, _ *http.Request, id string) (featurecost.QueryFeatureCostCSVRequest, error) {
			require.Equal(h.t, "feature-1", id)
			return featurecost.QueryFeatureCostCSVRequest{}, nil
		},
		func(context.Context, featurecost.QueryFeatureCostCSVRequest) (featurecost.QueryFeatureCostCSVResponse, error) {
			return featureCostRouteCSV{}, nil
		},
		commonhttp.CSVResponseEncoder[featurecost.QueryFeatureCostCSVResponse],
	)
}

type featureCostRouteCSV struct{}

func (featureCostRouteCSV) FileName() string    { return "feature-cost" }
func (featureCostRouteCSV) Records() [][]string { return [][]string{{"usage", "cost"}} }
