package openmeter

import (
	"io"
	"net/http"
	"testing"
)

func TestPtr(t *testing.T) {
	// The generic helper must work across types and return an independent pointer.
	if got := Ptr("x"); got == nil || *got != "x" {
		t.Fatalf("Ptr(\"x\") = %v", got)
	}
	if got := Ptr(42); got == nil || *got != 42 {
		t.Fatalf("Ptr(42) = %v", got)
	}
	if got := Ptr(true); got == nil || *got != true {
		t.Fatalf("Ptr(true) = %v", got)
	}
}

func TestDefaultUserAgentSent(t *testing.T) {
	// The default User-Agent must carry the SDK version so servers can attribute
	// traffic to a specific SDK build.
	var gotUA string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", contentTypeJSON)
		_, _ = io.WriteString(w, `{"id":"m1","key":"k","name":"n","aggregation":"sum","event_type":"e","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`)
	})

	if _, err := c.Meters.Get(t.Context(), "m1"); err != nil {
		t.Fatalf("Get: %v", err)
	}

	want := "openmeter-go-sdk/" + Version
	if gotUA != want {
		t.Fatalf("User-Agent = %q, want %q", gotUA, want)
	}
}
