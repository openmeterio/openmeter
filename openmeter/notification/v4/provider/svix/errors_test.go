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

	notificationprovider "github.com/openmeterio/openmeter/openmeter/notification/v4/provider"
)

func TestNormalizeTransportError(t *testing.T) {
	tests := []struct {
		name     string
		cause    error
		wantKind notificationprovider.ProviderErrorKind
		wantCode notificationprovider.ProviderErrorCode
	}{
		{
			name:     "connection refused",
			cause:    &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			wantKind: notificationprovider.ProviderErrorKindRetryable,
			wantCode: notificationprovider.ProviderErrorCodeUnavailable,
		},
		{
			name:     "HTTP client timeout",
			cause:    &url.Error{Op: "Get", URL: "http://svix", Err: context.DeadlineExceeded},
			wantKind: notificationprovider.ProviderErrorKindRetryable,
			wantCode: notificationprovider.ProviderErrorCodeUnavailable,
		},
		{
			name:     "EOF",
			cause:    io.EOF,
			wantKind: notificationprovider.ProviderErrorKindRetryable,
			wantCode: notificationprovider.ProviderErrorCodeUnavailable,
		},
		{
			name:     "unexpected EOF",
			cause:    io.ErrUnexpectedEOF,
			wantKind: notificationprovider.ProviderErrorKindRetryable,
			wantCode: notificationprovider.ProviderErrorCodeUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When normalizing a transport failure.
			err := normalizeError(tt.cause)

			// Then the failure is retryable and preserves its cause.
			var providerErr *notificationprovider.ProviderError

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
			providerErr := &notificationprovider.ProviderError{
				Kind:       notificationprovider.ProviderErrorKindRetryable,
				Code:       notificationprovider.ProviderErrorCodeRateLimited,
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
			require.Equal(t, notificationprovider.ProviderErrorKindRetryable, normalized.Kind)
			require.Equal(t, notificationprovider.ProviderErrorCodeRateLimited, normalized.Code)
			require.Equal(t, time.Second, *normalized.RetryAfter)
			require.Same(t, cause, normalized.Err)
			require.ErrorIs(t, normalized, cause)
		})
	}
}
