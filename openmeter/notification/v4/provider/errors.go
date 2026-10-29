package provider

import (
	"fmt"
	"time"
)

type ProviderErrorKind string

const (
	ProviderErrorKindRetryable     ProviderErrorKind = "retryable"
	ProviderErrorKindTerminal      ProviderErrorKind = "terminal"
	ProviderErrorKindConfiguration ProviderErrorKind = "configuration"
)

type ProviderErrorCode string

const (
	ProviderErrorCodeConfigurationInvalid ProviderErrorCode = "provider_configuration_invalid"
	ProviderErrorCodeResourceNotFound     ProviderErrorCode = "provider_resource_not_found"
	ProviderErrorCodeRequestFailed        ProviderErrorCode = "provider_request_failed"
	ProviderErrorCodeRateLimited          ProviderErrorCode = "provider_rate_limited"
	ProviderErrorCodeUnavailable          ProviderErrorCode = "provider_unavailable"
)

// ProviderError carries a normalized classification and an internal cause.
// Notification sanitizes public messages rather than exposing Err to callers.
// Operation errors may wrap it; callers use errors.As to inspect classification
// and errors.Is to detect cancellation independently of provider failures.
type ProviderError struct {
	Kind       ProviderErrorKind
	Code       ProviderErrorCode
	RetryAfter *time.Duration
	Err        error
}

var _ error = (*ProviderError)(nil)

func (e *ProviderError) Error() string {
	if e.Err == nil {
		return string(e.Code)
	}

	return fmt.Sprintf("%s: %s", e.Code, e.Err)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}
