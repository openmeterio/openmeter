package openmeter_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	openmeter "github.com/openmeterio/openmeter/api/v3/client"
)

func TestFeatureCostCSV(t *testing.T) {
	const csv = "from,to,usage,cost,currency,detail\n2024-01-01T00:00:00Z,2024-01-02T00:00:00Z,100,0.001,USD,\n"
	for _, stream := range []bool{false, true} {
		name := "buffered"
		if stream {
			name = "streamed"
		}
		t.Run(name, func(t *testing.T) {
			om := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/openmeter/features/feature-1/cost/query" {
					t.Errorf("unexpected route: %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Accept") != "text/csv" || r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("unexpected content negotiation: %v", r.Header)
				}
				var body struct {
					GroupByDimensions []string `json:"group_by_dimensions"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode query: %v", err)
				} else if len(body.GroupByDimensions) != 1 || body.GroupByDimensions[0] != "model" {
					t.Errorf("group_by_dimensions = %v, want [model]", body.GroupByDimensions)
				}
				w.Header().Set("Content-Type", "text/csv")
				_, _ = io.WriteString(w, csv)
			}))
			request := openmeter.MeterQueryRequest{GroupByDimensions: &[]string{"model"}}
			var data []byte
			var err error
			if stream {
				body, streamErr := om.Features.QueryCostCSVStream(t.Context(), "feature-1", request)
				if streamErr != nil {
					t.Fatalf("QueryCostCSVStream: %v", streamErr)
				}
				defer body.Close()
				data, err = io.ReadAll(body)
			} else {
				data, err = om.Features.QueryCostCSV(t.Context(), "feature-1", request)
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != csv {
				t.Errorf("CSV = %q, want %q", data, csv)
			}
		})
	}
}
