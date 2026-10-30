package svix

import (
	"errors"
	"io"
	"net"
	"net/http"

	svix "github.com/svix/svix-webhooks/v2/go"

	notificationprovider "github.com/openmeterio/openmeter/openmeter/notification/v4/provider"
)

func normalizeError(err error) *notificationprovider.ProviderError {
	if providerErr, ok := errors.AsType[*notificationprovider.ProviderError](err); ok {
		return providerErr
	}

	normalized := &notificationprovider.ProviderError{
		Kind: notificationprovider.ProviderErrorKindTerminal,
		Code: notificationprovider.ProviderErrorCodeRequestFailed,
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
			normalized.Code = notificationprovider.ProviderErrorCodeResourceNotFound
		case status == http.StatusBadRequest, status == http.StatusUnauthorized,
			status == http.StatusForbidden, status == http.StatusUnprocessableEntity:
			normalized.Kind = notificationprovider.ProviderErrorKindConfiguration
			normalized.Code = notificationprovider.ProviderErrorCodeConfigurationInvalid
		case status == http.StatusTooManyRequests:
			normalized.Kind = notificationprovider.ProviderErrorKindRetryable
			normalized.Code = notificationprovider.ProviderErrorCodeRateLimited
		case status == http.StatusRequestTimeout, status >= http.StatusInternalServerError:
			normalized.Kind = notificationprovider.ProviderErrorKindRetryable
			normalized.Code = notificationprovider.ProviderErrorCodeUnavailable
		}
	case errors.As(err, &networkErr), errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		normalized.Kind = notificationprovider.ProviderErrorKindRetryable
		normalized.Code = notificationprovider.ProviderErrorCodeUnavailable
	}

	// The pinned SDK exposes status and body, but not Retry-After headers.
	// Leave RetryAfter unset so the reconciler can apply its default backoff.
	return normalized
}
