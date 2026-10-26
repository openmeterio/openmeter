package svix

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/notification/v4"
)

func TestNormalizeTransportError(t *testing.T) {
	tests := []struct {
		name     string
		cause    error
		wantKind v4.ProviderErrorKind
		wantCode v4.ProviderErrorCode
	}{
		{
			name:     "connection refused",
			cause:    &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			wantKind: v4.ProviderErrorKindRetryable,
			wantCode: v4.ProviderErrorCodeUnavailable,
		},
		{
			name:     "HTTP client timeout",
			cause:    &url.Error{Op: "Get", URL: "http://svix", Err: context.DeadlineExceeded},
			wantKind: v4.ProviderErrorKindRetryable,
			wantCode: v4.ProviderErrorCodeUnavailable,
		},
		{
			name:     "EOF",
			cause:    io.EOF,
			wantKind: v4.ProviderErrorKindRetryable,
			wantCode: v4.ProviderErrorCodeUnavailable,
		},
		{
			name:     "unexpected EOF",
			cause:    io.ErrUnexpectedEOF,
			wantKind: v4.ProviderErrorKindRetryable,
			wantCode: v4.ProviderErrorCodeUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When normalizing a transport failure.
			err := normalizeError(tt.cause)

			// Then the failure is retryable and preserves its cause.
			var providerErr *v4.ProviderError

			require.ErrorAs(t, err, &providerErr)
			require.Equal(t, tt.wantKind, providerErr.Kind)
			require.Equal(t, tt.wantCode, providerErr.Code)
			require.ErrorIs(t, err, tt.cause)
		})
	}
}

func TestNormalizeProviderError(t *testing.T) {
	tests := []struct {
		name string
		wrap bool
	}{
		{name: "direct"},
		{name: "wrapped", wrap: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given an already classified failure with provider retry metadata.
			cause := errors.New("rate limit exceeded")
			providerErr := &v4.ProviderError{
				Kind:       v4.ProviderErrorKindRetryable,
				Code:       v4.ProviderErrorCodeRateLimited,
				RetryAfter: new(time.Second),
				Err:        cause,
			}
			var input error = providerErr

			if tt.wrap {
				input = fmt.Errorf("failed to get endpoint: %w", input)
			}

			// When classification is requested again.
			normalized := normalizeError(input)

			// Then the existing error is preserved without copying or mutation.
			require.Same(t, providerErr, normalized)
			require.Equal(t, v4.ProviderErrorKindRetryable, normalized.Kind)
			require.Equal(t, v4.ProviderErrorCodeRateLimited, normalized.Code)
			require.Equal(t, time.Second, *normalized.RetryAfter)
			require.Same(t, cause, normalized.Err)
			require.ErrorIs(t, normalized, cause)
		})
	}
}
