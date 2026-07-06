package openmeter

import (
	"context"
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
			got, err := retryIdempotentOnly(context.Background(), resp, nil)
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
	// No response (resp == nil): the method is unknown and the request most
	// likely never reached the server, so the default policy applies and a
	// transport error is retried.
	got, err := retryIdempotentOnly(context.Background(), nil, errors.New("connection refused"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got {
		t.Fatal("transport error should be retried, got no retry")
	}
}
