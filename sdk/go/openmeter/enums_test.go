package openmeter

import (
	"encoding/json"
	"strings"
	"testing"
)

// A server may introduce enum values after this SDK was built. Those values must
// decode into the string-typed enum unchanged and re-encode as-is, so an older
// SDK binary keeps working against a newer server. These tests pin that
// forward-compatibility contract for the SDK's enums.
func TestEnums_UnknownValuePreservedOnRoundTrip(t *testing.T) {
	t.Run("MeterAggregation", func(t *testing.T) {
		// "median" is deliberately not one of the MeterAggregation constants.
		const raw = `{"id":"m1","key":"k","name":"n","aggregation":"median","event_type":"e","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`

		var m Meter
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		if m.Aggregation != MeterAggregation("median") {
			t.Fatalf("aggregation = %q, want %q (unknown value not preserved)", m.Aggregation, "median")
		}

		out, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if !strings.Contains(string(out), `"aggregation":"median"`) {
			t.Fatalf("re-encoded meter lost the unknown aggregation: %s", out)
		}
	})

	t.Run("MeterQueryGranularity", func(t *testing.T) {
		// "PT5M" is a valid ISO-8601 duration but not one of the granularity
		// constants this SDK defines.
		const raw = `{"granularity":"PT5M"}`

		var req MeterQueryRequest
		if err := json.Unmarshal([]byte(raw), &req); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}

		if req.Granularity == nil || *req.Granularity != MeterQueryGranularity("PT5M") {
			t.Fatalf("granularity = %v, want %q (unknown value not preserved)", req.Granularity, "PT5M")
		}

		out, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if !strings.Contains(string(out), `"granularity":"PT5M"`) {
			t.Fatalf("re-encoded request lost the unknown granularity: %s", out)
		}
	})
}
