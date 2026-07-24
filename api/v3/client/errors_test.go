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

func TestAPIErrorInvalidParameters(t *testing.T) {
	t.Parallel()

	getAPIError := func(t *testing.T, body string) *openmeter.APIError {
		t.Helper()

		om := newTestClient(t, (&requestRecorder{}).handler(http.StatusBadRequest, body))
		_, err := om.Meters.Get(t.Context(), "m-1")

		apiErr, ok := openmeter.AsAPIError(err)
		if !ok {
			t.Fatalf("error %v is not an *APIError", err)
		}
		return apiErr
	}

	t.Run("standard rule", func(t *testing.T) {
		t.Parallel()

		apiErr := getAPIError(t, `{"status":400,"title":"Bad Request","invalid_parameters":[{"field":"name","rule":"required","reason":"is required","source":"body"}]}`)

		if len(apiErr.InvalidParameters) != 1 {
			t.Fatalf("InvalidParameters = %v, want one entry", apiErr.InvalidParameters)
		}
		p := apiErr.InvalidParameters[0]
		if p.Field != "name" || p.Rule != openmeter.InvalidRuleRequired || p.Reason != "is required" || p.Source != openmeter.InvalidParameterSourceBody {
			t.Errorf("unexpected parameter: %+v", p)
		}
		if !p.Rule.Valid() {
			t.Error("Rule.Valid() = false for a known rule")
		}
		if !p.Source.Valid() {
			t.Error("Source.Valid() = false for a known source")
		}
	})

	t.Run("minimum and maximum rules", func(t *testing.T) {
		t.Parallel()

		apiErr := getAPIError(t, `{"status":400,"title":"Bad Request","invalid_parameters":[{"field":"key","rule":"min_length","reason":"too short","minimum":5},{"field":"key","rule":"max_length","reason":"too long","maximum":100}]}`)

		if len(apiErr.InvalidParameters) != 2 {
			t.Fatalf("InvalidParameters = %v, want two entries", apiErr.InvalidParameters)
		}
		if p := apiErr.InvalidParameters[0]; p.Minimum == nil || *p.Minimum != 5 {
			t.Errorf("Minimum = %v, want 5", p.Minimum)
		}
		if p := apiErr.InvalidParameters[1]; p.Maximum == nil || *p.Maximum != 100 {
			t.Errorf("Maximum = %v, want 100", p.Maximum)
		}
	})

	t.Run("choice rule with mixed value types", func(t *testing.T) {
		t.Parallel()

		apiErr := getAPIError(t, `{"status":400,"title":"Bad Request","invalid_parameters":[{"field":"kind","rule":"enum","reason":"not a choice","choices":["a","b",3]}]}`)

		if len(apiErr.InvalidParameters) != 1 {
			t.Fatalf("InvalidParameters = %v, want one entry", apiErr.InvalidParameters)
		}
		choices := apiErr.InvalidParameters[0].Choices
		if len(choices) != 3 {
			t.Fatalf("Choices = %v, want three values", choices)
		}
		if choices[0] != "a" || choices[1] != "b" {
			t.Errorf("Choices = %v, want string values decoded verbatim", choices)
		}
		if n, ok := choices[2].(float64); !ok || n != 3 {
			t.Errorf("Choices[2] = %v, want the numeric choice to decode", choices[2])
		}
	})

	t.Run("dependent rule", func(t *testing.T) {
		t.Parallel()

		apiErr := getAPIError(t, `{"status":400,"title":"Bad Request","invalid_parameters":[{"field":"start","rule":"dependent_fields","reason":"requires end","dependents":["end"]}]}`)

		if len(apiErr.InvalidParameters) != 1 {
			t.Fatalf("InvalidParameters = %v, want one entry", apiErr.InvalidParameters)
		}
		p := apiErr.InvalidParameters[0]
		if p.Rule != openmeter.InvalidRuleDependentFields || len(p.Dependents) != 1 || p.Dependents[0] != "end" {
			t.Errorf("unexpected parameter: %+v", p)
		}
	})

	t.Run("unknown rule round-trips", func(t *testing.T) {
		t.Parallel()

		apiErr := getAPIError(t, `{"status":400,"title":"Bad Request","invalid_parameters":[{"field":"x","rule":"some_future_rule","reason":"new server"}]}`)

		p := apiErr.InvalidParameters[0]
		if p.Rule != openmeter.InvalidRule("some_future_rule") {
			t.Errorf("Rule = %q, want the unknown value preserved verbatim", p.Rule)
		}
		if p.Rule.Valid() {
			t.Error("Rule.Valid() = true for an unknown rule")
		}
	})

	t.Run("malformed invalid_parameters keeps sibling fields", func(t *testing.T) {
		t.Parallel()

		apiErr := getAPIError(t, `{"status":400,"title":"Bad Request","detail":"still here","invalid_parameters":{"not":"an array"}}`)

		if apiErr.Status != 400 || apiErr.Title != "Bad Request" || apiErr.Detail != "still here" {
			t.Errorf("sibling fields did not survive a malformed invalid_parameters: %+v", apiErr)
		}
		if apiErr.InvalidParameters != nil {
			t.Errorf("InvalidParameters = %v, want nil for a malformed field", apiErr.InvalidParameters)
		}
	})

	t.Run("absent invalid_parameters stays nil", func(t *testing.T) {
		t.Parallel()

		apiErr := getAPIError(t, `{"status":400,"title":"Bad Request"}`)

		if apiErr.InvalidParameters != nil {
			t.Errorf("InvalidParameters = %v, want nil when absent", apiErr.InvalidParameters)
		}
	})
}
