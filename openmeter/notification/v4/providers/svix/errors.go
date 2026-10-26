package svix

import (
	"errors"
	"io"
	"net"
	"net/http"

	svix "github.com/svix/svix-webhooks/v2/go"

	"github.com/openmeterio/openmeter/openmeter/notification/v4"
)

func normalizeError(err error) *v4.ProviderError {
	if providerErr, ok := errors.AsType[*v4.ProviderError](err); ok {
		return providerErr
	}

	normalized := &v4.ProviderError{
		Kind: v4.ProviderErrorKindTerminal,
		Code: v4.ProviderErrorCodeRequestFailed,
		Err:  err,
	}

	var (
		svixErr    *svix.Error
		networkErr net.Error
	)

	switch {
	case errors.As(err, &svixErr):
		switch status := svixErr.Status(); {
		case status == http.StatusNotFound:
			normalized.Code = v4.ProviderErrorCodeResourceNotFound
		case status == http.StatusBadRequest, status == http.StatusUnauthorized,
			status == http.StatusForbidden, status == http.StatusUnprocessableEntity:
			normalized.Kind = v4.ProviderErrorKindConfiguration
			normalized.Code = v4.ProviderErrorCodeConfigurationInvalid
		case status == http.StatusTooManyRequests:
			normalized.Kind = v4.ProviderErrorKindRetryable
			normalized.Code = v4.ProviderErrorCodeRateLimited
		case status == http.StatusRequestTimeout, status >= http.StatusInternalServerError:
			normalized.Kind = v4.ProviderErrorKindRetryable
			normalized.Code = v4.ProviderErrorCodeUnavailable
		}
	case errors.As(err, &networkErr), errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		normalized.Kind = v4.ProviderErrorKindRetryable
		normalized.Code = v4.ProviderErrorCodeUnavailable
	}

	// The pinned SDK exposes status and body, but not Retry-After headers.
	// Leave RetryAfter unset so the reconciler can apply its default backoff.
	return normalized
}
