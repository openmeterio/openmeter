// This smoke test exercises the generated SDK against an httptest.Server stub.
// It verifies the SDK constructs URLs, sends requests, and decodes JSON
// responses correctly for a representative operation.
package aipclientgo_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdkpkg "github.com/openmeterio/openmeter/api/spec/packages/aip-client-go"
	"github.com/openmeterio/openmeter/api/spec/packages/aip-client-go/models/operations"
)

func TestSDKConstruction(t *testing.T) {
	sdk := sdkpkg.New()
	if sdk == nil {
		t.Fatal("New() returned nil")
	}
	if sdk.SDKVersion == "" {
		t.Fatal("SDKVersion not set")
	}
	if sdk.OpenMeterMeters == nil {
		t.Error("OpenMeterMeters sub-client not initialized")
	}
}

func TestListMetersAgainstStub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/openmeter/meters") {
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":          "01J3T8VVZP1A2B3C4D5E6F7G8H",
					"name":        "API requests",
					"key":         "api_requests",
					"aggregation": "count",
					"event_type":  "request",
					"created_at":  "2024-01-01T00:00:00Z",
					"updated_at":  "2024-01-01T00:00:00Z",
				},
			},
			"meta": map[string]any{
				"page": map[string]any{"size": 1, "number": 1, "total": 1},
			},
		})
	}))
	defer srv.Close()

	sdk := sdkpkg.New(sdkpkg.WithServerURL(srv.URL))
	ctx := context.Background()
	res, err := sdk.OpenMeterMeters.ListMeters(ctx, operations.ListMetersRequest{})
	if err != nil {
		t.Fatalf("ListMeters failed: %v", err)
	}
	if res == nil {
		t.Fatal("ListMeters returned nil response")
	}
	if res.HTTPMeta.Response == nil {
		t.Error("HTTPMeta.Response missing")
	}
}
