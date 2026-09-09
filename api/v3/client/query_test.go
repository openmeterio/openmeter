// Hand-written query-serialization tests for the generated OpenMeter Go SDK.
// The generator's output cleaner preserves *_test.go files, so these survive
// regeneration. The recorder harness lives in transport_test.go.
package openmeter_test

import (
	"net/http"
	"testing"
	"time"

	openmeter "github.com/openmeterio/openmeter/api/v3/client"
)

func TestFilterQuerySerialization(t *testing.T) {
	t.Parallel()

	rec := &requestRecorder{}
	om := newTestClient(t, rec.handler(http.StatusOK, emptyPageBody))

	params := openmeter.MeterListParams{Filter: &openmeter.MeterFilter{
		Key:  &openmeter.StringFilter{Oeq: []string{"tokens", "requests"}},
		Name: &openmeter.StringFilter{Contains: openmeter.String("gpt")},
	}}
	if _, err := om.Meters.List(t.Context(), params); err != nil {
		t.Fatalf("Meters.List: %v", err)
	}

	q := rec.last(t).query
	if got := q.Get("filter[key][oeq]"); got != "tokens,requests" {
		t.Errorf("filter[key][oeq] = %q, want %q", got, "tokens,requests")
	}
	if got := q.Get("filter[name][contains]"); got != "gpt" {
		t.Errorf("filter[name][contains] = %q, want %q", got, "gpt")
	}
}

func TestPresenceFilterQuerySerialization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		exists bool
		want   string
	}{
		{name: "present", exists: true, want: ""},
		{name: "null", exists: false, want: "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := &requestRecorder{}
			om := newTestClient(t, rec.handler(http.StatusOK, emptyPageBody))
			params := openmeter.ChargeListParams{Filter: &openmeter.ChargeFilter{
				ValidationIssues: &openmeter.PresenceFilter{Exists: openmeter.Bool(tt.exists)},
			}}
			if _, err := om.Customers.Charges.List(t.Context(), "customer-id", params); err != nil {
				t.Fatalf("Customers.Charges.List: %v", err)
			}

			q := rec.last(t).query
			if !q.Has("filter[validation_issues]") {
				t.Fatal("filter[validation_issues] is missing")
			}
			if got := q.Get("filter[validation_issues]"); got != tt.want {
				t.Errorf("filter[validation_issues] = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScalarQueryParamSerialization(t *testing.T) {
	t.Parallel()

	rec := &requestRecorder{}
	om := newTestClient(t, rec.handler(http.StatusOK, "{}"))

	timestamp := time.Date(2026, 5, 11, 10, 30, 0, 0, time.UTC)
	params := openmeter.GetCustomerCreditBalanceParams{Timestamp: openmeter.Time(timestamp)}
	if _, err := om.Customers.Credits.Balance.Get(t.Context(), "cus-1", params); err != nil {
		t.Fatalf("Customers.Credits.Balance.Get: %v", err)
	}

	if got := rec.last(t).query.Get("timestamp"); got != "2026-05-11T10:30:00Z" {
		t.Errorf("timestamp = %q, want %q", got, "2026-05-11T10:30:00Z")
	}
}
