// Hand-written wire tests for the generated OpenMeter Go SDK. The generator's
// output cleaner preserves *_test.go files, so these survive regeneration.
package openmeter_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	openmeter "github.com/openmeterio/openmeter/api/v3/client"
)

func TestAPIErrorParsesRFC7807(t *testing.T) {
	t.Parallel()

	body := `{"status":404,"type":"https://openmeter.io/problems/not-found","title":"Not Found","detail":"meter not found","instance":"kong:trace:abc123"}`
	rec := &requestRecorder{}
	om := newTestClient(t, rec.handler(http.StatusNotFound, body))

	_, err := om.Meters.Get(t.Context(), "m-1")
	apiErr, ok := openmeter.AsAPIError(err)
	if !ok {
		t.Fatalf("error %v is not an *APIError", err)
	}

	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Status != 404 {
		t.Errorf("Status = %d, want 404", apiErr.Status)
	}
	if apiErr.Type != "https://openmeter.io/problems/not-found" {
		t.Errorf("Type = %q, want the problem type", apiErr.Type)
	}
	if apiErr.Title != "Not Found" {
		t.Errorf("Title = %q, want %q", apiErr.Title, "Not Found")
	}
	if apiErr.Detail != "meter not found" {
		t.Errorf("Detail = %q, want %q", apiErr.Detail, "meter not found")
	}
	if apiErr.Instance != "kong:trace:abc123" {
		t.Errorf("Instance = %q, want %q", apiErr.Instance, "kong:trace:abc123")
	}
	if string(apiErr.RawBody) != body {
		t.Errorf("RawBody = %q, want the undecoded body", apiErr.RawBody)
	}
	if want := "openmeter: 404 Not Found: meter not found"; apiErr.Error() != want {
		t.Errorf("Error() = %q, want %q", apiErr.Error(), want)
	}
}

func TestAPIErrorTitleOnly(t *testing.T) {
	t.Parallel()

	om := newTestClient(t, (&requestRecorder{}).handler(http.StatusTeapot, `{"status":418,"title":"Teapot"}`))

	_, err := om.Meters.Get(t.Context(), "m-1")
	apiErr, ok := openmeter.AsAPIError(err)
	if !ok {
		t.Fatalf("error %v is not an *APIError", err)
	}
	if want := "openmeter: 418 Teapot"; apiErr.Error() != want {
		t.Errorf("Error() = %q, want %q", apiErr.Error(), want)
	}
}

func TestAPIErrorNonJSONBodyFallsBackToRawEcho(t *testing.T) {
	t.Parallel()

	t.Run("short body echoed in full", func(t *testing.T) {
		body := "<html>bad gateway</html>"
		om := newTestClient(t, (&requestRecorder{}).handler(http.StatusBadGateway, body))

		_, err := om.Meters.Get(t.Context(), "m-1")
		apiErr, ok := openmeter.AsAPIError(err)
		if !ok {
			t.Fatalf("error %v is not an *APIError", err)
		}
		if apiErr.Title != "" {
			t.Errorf("Title = %q, want empty for a non-problem body", apiErr.Title)
		}
		if want := "openmeter: unexpected status 502: " + body; apiErr.Error() != want {
			t.Errorf("Error() = %q, want %q", apiErr.Error(), want)
		}
	})

	t.Run("long body truncated at 512 bytes", func(t *testing.T) {
		body := strings.Repeat("x", 600)
		om := newTestClient(t, (&requestRecorder{}).handler(http.StatusBadGateway, body))

		_, err := om.Meters.Get(t.Context(), "m-1")
		apiErr, ok := openmeter.AsAPIError(err)
		if !ok {
			t.Fatalf("error %v is not an *APIError", err)
		}

		want := "openmeter: unexpected status 502: " + strings.Repeat("x", 512) + "… (truncated)"
		if apiErr.Error() != want {
			t.Errorf("Error() = %q, want %q", apiErr.Error(), want)
		}
		// The message is truncated; RawBody still carries the full payload.
		if len(apiErr.RawBody) != 600 {
			t.Errorf("len(RawBody) = %d, want the full 600 bytes", len(apiErr.RawBody))
		}
	})
}

func TestAsAPIError(t *testing.T) {
	t.Parallel()

	om := newTestClient(t, (&requestRecorder{}).handler(http.StatusInternalServerError, `{"status":500,"title":"Boom"}`))

	_, err := om.Meters.Get(t.Context(), "m-1")
	if err == nil {
		t.Fatal("Meters.Get returned nil error for a 500 response")
	}

	wrapped := fmt.Errorf("listing usage: %w", err)
	apiErr, ok := openmeter.AsAPIError(wrapped)
	if !ok {
		t.Fatalf("AsAPIError did not find the APIError inside %v", wrapped)
	}
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}

	if got, ok := openmeter.AsAPIError(errors.New("plain")); ok || got != nil {
		t.Errorf("AsAPIError(plain error) = (%v, %t), want (nil, false)", got, ok)
	}
}

func TestDecodeAPIError(t *testing.T) {
	t.Parallel()

	type validationProblem struct {
		Status int    `json:"status"`
		Title  string `json:"title"`
		Errors []struct {
			Field string `json:"field"`
		} `json:"errors"`
	}

	t.Run("decodes a typed error body", func(t *testing.T) {
		body := `{"status":400,"title":"Bad Request","errors":[{"field":"key"},{"field":"name"}]}`
		om := newTestClient(t, (&requestRecorder{}).handler(http.StatusBadRequest, body))

		_, err := om.Meters.Get(t.Context(), "m-1")
		problem, ok, decodeErr := openmeter.DecodeAPIError[validationProblem](err)
		if decodeErr != nil {
			t.Fatalf("DecodeAPIError: %v", decodeErr)
		}
		if !ok {
			t.Fatal("DecodeAPIError reported the error is not an APIError")
		}
		if problem.Status != 400 || problem.Title != "Bad Request" || len(problem.Errors) != 2 || problem.Errors[1].Field != "name" {
			t.Errorf("decoded problem = %+v, want the typed body", problem)
		}
	})

	t.Run("non-API errors are reported as not decodable", func(t *testing.T) {
		problem, ok, decodeErr := openmeter.DecodeAPIError[validationProblem](errors.New("dial tcp: refused"))
		if ok || decodeErr != nil {
			t.Errorf("DecodeAPIError(plain error) = (%+v, %t, %v), want ok=false with nil error", problem, ok, decodeErr)
		}
	})

	t.Run("undecodable body surfaces the decode error", func(t *testing.T) {
		om := newTestClient(t, (&requestRecorder{}).handler(http.StatusInternalServerError, "not json"))

		_, err := om.Meters.Get(t.Context(), "m-1")
		_, ok, decodeErr := openmeter.DecodeAPIError[validationProblem](err)
		if !ok {
			t.Fatal("DecodeAPIError reported the error is not an APIError")
		}
		if decodeErr == nil {
			t.Error("DecodeAPIError returned nil error for a non-JSON body")
		}
	})
}

func TestEmptyIDGuard(t *testing.T) {
	t.Parallel()

	// Empty IDs must be rejected client-side: no request may reach the server.
	om := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected HTTP request %s %s for an empty-ID call", r.Method, r.URL)
	}))

	cases := []struct {
		name      string
		wantParam string
		call      func(ctx context.Context) error
	}{
		{
			name:      "meters get",
			wantParam: "meterID",
			call: func(ctx context.Context) error {
				_, err := om.Meters.Get(ctx, "")
				return err
			},
		},
		{
			name:      "meters update",
			wantParam: "meterID",
			call: func(ctx context.Context) error {
				_, err := om.Meters.Update(ctx, "", openmeter.UpdateMeterRequest{})
				return err
			},
		},
		{
			name:      "meters delete",
			wantParam: "meterID",
			call:      func(ctx context.Context) error { return om.Meters.Delete(ctx, "") },
		},
		{
			name:      "customers get",
			wantParam: "customerID",
			call: func(ctx context.Context) error {
				_, err := om.Customers.Get(ctx, "")
				return err
			},
		},
		{
			name:      "invoices get",
			wantParam: "invoiceID",
			call: func(ctx context.Context) error {
				_, err := om.Invoices.Get(ctx, "")
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call(t.Context())
			if err == nil {
				t.Fatal("call with empty ID returned nil error")
			}
			if !errors.Is(err, openmeter.ErrEmptyID) {
				t.Errorf("errors.Is(err, ErrEmptyID) = false for %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantParam) {
				t.Errorf("error %q does not name the parameter %q", err, tc.wantParam)
			}
		})
	}
}
