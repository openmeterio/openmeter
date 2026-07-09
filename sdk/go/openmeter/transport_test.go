package openmeter

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRetryIdempotentOnly(t *testing.T) {
	tests := []struct {
		name   string
		method string
		status int // 0 means "no response, transport error"
		want   bool
	}{
		// Idempotent methods retry on retryable server responses.
		{"GET 500 retries", http.MethodGet, http.StatusInternalServerError, true},
		{"HEAD 503 retries", http.MethodHead, http.StatusServiceUnavailable, true},
		{"GET 429 retries", http.MethodGet, http.StatusTooManyRequests, true},

		// Non-idempotent methods are never retried once a response arrives,
		// even on 5xx, to avoid duplicating a side effect the server may have
		// already applied.
		{"POST 500 does not retry", http.MethodPost, http.StatusInternalServerError, false},
		{"PUT 503 does not retry", http.MethodPut, http.StatusServiceUnavailable, false},
		{"DELETE 500 does not retry", http.MethodDelete, http.StatusInternalServerError, false},
		{"PATCH 429 does not retry", http.MethodPatch, http.StatusTooManyRequests, false},

		// Success responses never retry regardless of method.
		{"GET 200 does not retry", http.MethodGet, http.StatusOK, false},
		{"POST 200 does not retry", http.MethodPost, http.StatusOK, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.status,
				Request:    httptest.NewRequest(tt.method, "/", nil),
			}
			got, err := retryIdempotentOnly(t.Context(), resp, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("retry = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetryIdempotentOnly_TransportError(t *testing.T) {
	// No response (resp == nil): the method comes from the request context. A
	// connection can drop after the server processed the request, so a
	// non-idempotent method must not be retried even without a response.
	transportErr := errors.New("connection refused")

	tests := []struct {
		name   string
		method string // empty simulates a non-SDK caller: no method on the context
		want   bool
	}{
		{"POST transport error does not retry", http.MethodPost, false},
		{"DELETE transport error does not retry", http.MethodDelete, false},
		{"GET transport error retries", http.MethodGet, true},
		// No method on the context (non-SDK caller): fall back to the default policy.
		{"unknown method retries", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			if tt.method != "" {
				ctx = withRequestMethod(ctx, tt.method)
			}

			got, err := retryIdempotentOnly(ctx, nil, transportErr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("retry = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRequest_CarriesMethodOnContext(t *testing.T) {
	// retryIdempotentOnly reads the method from the request context on a
	// transport error; newRequest must put it there. retryablehttp passes
	// req.Context() to the retry policy, so this is what closes the loop.
	c, err := New("https://example.com/api/v3")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := c.newRequest(t.Context(), http.MethodPost, "/openmeter/meters", nil, nil, contentTypeJSON)
	if err != nil {
		t.Fatalf("newRequest: %v", err)
	}

	if got := methodFromContext(req.Context()); got != http.MethodPost {
		t.Fatalf("method on request context = %q, want %q", got, http.MethodPost)
	}
}
