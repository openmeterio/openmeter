package apierrors

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

const (
	// Content Type
	ContentTypeKey          = "Content-Type"
	ContentTypeProblemValue = "application/problem+json"

	// Internal
	InternalType   = "https://kongapi.info/konnect/internal"
	InternalTitle  = "Internal"
	InternalDetail = "An internal failure occurred"

	// Service Unavailable
	UnavailableType   = "https://kongapi.info/konnect/unavailable"
	UnavailableTitle  = "Unavailable"
	UnavailableDetail = "The requested service is unavailable"

	// NotImplemented
	NotImplementedType   = "https://kongapi.info/konnect/not-implemented"
	NotImplementedTitle  = "Not Implemented"
	NotImplementedDetail = "The requested functionality is not implemented"

	// Unauthenticated
	UnauthenticatedType   = "https://kongapi.info/konnect/unauthenticated"
	UnauthenticatedTitle  = "Unauthenticated"
	UnauthenticatedDetail = "A valid token is required"

	// Forbidden
	ForbiddenType   = "https://kongapi.info/konnect/unauthorized"
	ForbiddenTitle  = "Forbidden"
	ForbiddenDetail = "Permission denied"

	// NotFound
	NotFoundType    = "https://kongapi.info/konnect/not-found"
	NotFoundTitle   = "Not Found"
	NotFoundDetail  = "The requested %s was not found"
	NotFoundDetails = "The requested %s were not found: %v"

	// Method Not Allowed
	MethodNotAllowedType   = "https://kongapi.info/konnect/method-not-allowed"
	MethodNotAllowedTitle  = "Method Not Allowed"
	MethodNotAllowedDetail = "The requested method is not allowed"

	// BadRequest
	BadRequestType  = "https://kongapi.info/konnect/bad-request"
	BadRequestTitle = "Bad Request"

	// Precondition Failed
	PreconditionFailedType  = "https://kongapi.info/konnect/precondition-failed"
	PreconditionFailedTitle = "Precondition Failed"

	// Rate Limit
	RateLimitType   = "https://kongapi.info/konnect/rate-limited"
	RateLimitTitle  = "Rate Limited"
	RateLimitDetail = "Too many requests"

	// Conflict
	ConflictType  = "https://kongapi.info/konnect/resource-conflict"
	ConflictTitle = "Conflict"

	// Empty Set
	EmptySetType       = "Empty Set"
	EmptySetCursorType = "Empty Set Cursor"
)

// NewInternalError generates a not found error.
func NewInternalError(ctx context.Context, err error) *BaseAPIError {
	return &BaseAPIError{
		Type:            InternalType,
		Status:          http.StatusInternalServerError,
		Title:           InternalTitle,
		Instance:        instance(ctx),
		Detail:          InternalDetail,
		UnderlyingError: err,
		ctx:             ctx,
	}
}

// NewServiceUnavailable generates a not found error.
func NewServiceUnavailable(ctx context.Context, err error) *BaseAPIError {
	return &BaseAPIError{
		Type:            InternalType,
		Status:          http.StatusServiceUnavailable,
		Title:           InternalTitle,
		Instance:        instance(ctx),
		Detail:          InternalDetail,
		UnderlyingError: err,
		ctx:             ctx,
	}
}

// NewUnauthenticatedError generates an unauthenticated error.
func NewUnauthenticatedError(ctx context.Context, err error) *BaseAPIError {
	return &BaseAPIError{
		Type:            UnauthenticatedType,
		Status:          http.StatusUnauthorized,
		Title:           UnauthenticatedTitle,
		Instance:        instance(ctx),
		Detail:          UnauthenticatedDetail,
		UnderlyingError: err,
		ctx:             ctx,
	}
}

// NewForbiddenError generates an unauthorized error.
func NewForbiddenError(ctx context.Context, err error) *BaseAPIError {
	return &BaseAPIError{
		Type:            ForbiddenType,
		Status:          http.StatusForbidden,
		Title:           ForbiddenTitle,
		Instance:        instance(ctx),
		Detail:          ForbiddenDetail,
		UnderlyingError: err,
		ctx:             ctx,
	}
}

// NewForbiddenErrorDetail generates an forbidden error with a user readable/detailed message.
func NewForbiddenErrorDetail(ctx context.Context, detailMessage string) *BaseAPIError {
	return &BaseAPIError{
		Type:     ForbiddenType,
		Status:   http.StatusForbidden,
		Title:    ForbiddenDetail,
		Instance: instance(ctx),
		Detail:   MakeSentenceCase(detailMessage),
		ctx:      ctx,
	}
}

// NewNotFoundError generates a not found error.
func NewNotFoundError(ctx context.Context, err error, entityType string) *BaseAPIError {
	if entityType != "" {
		return &BaseAPIError{
			Type:            NotFoundType,
			Status:          http.StatusNotFound,
			Title:           NotFoundTitle,
			Instance:        instance(ctx),
			Detail:          fmt.Sprintf(NotFoundDetail, entityType),
			UnderlyingError: err,
			ctx:             ctx,
		}
	}
	return &BaseAPIError{
		Type:            NotFoundType,
		Status:          http.StatusNotFound,
		Title:           NotFoundTitle,
		Instance:        instance(ctx),
		UnderlyingError: err,
		ctx:             ctx,
	}
}

// NewNotFoundErrors generates a not found error for multiple resources.
func NewNotFoundErrors(
	ctx context.Context,
	err error,
	entityType string,
	resources any,
) *BaseAPIError {
	return &BaseAPIError{
		Type:            NotFoundType,
		Status:          http.StatusNotFound,
		Title:           NotFoundTitle,
		Instance:        instance(ctx),
		Detail:          fmt.Sprintf(NotFoundDetails, entityType, resources),
		UnderlyingError: err,
		ctx:             ctx,
	}
}

// NewMethodNotAllowedError generates a method not allowed error.
func NewMethodNotAllowedError(ctx context.Context) *BaseAPIError {
	return &BaseAPIError{
		Type:     MethodNotAllowedType,
		Status:   http.StatusMethodNotAllowed,
		Title:    MethodNotAllowedTitle,
		Instance: instance(ctx),
		Detail:   MethodNotAllowedDetail,
		ctx:      ctx,
	}
}

// NewBadRequestError generates a bad request error.
func NewBadRequestError(ctx context.Context, err error, invalidFields InvalidParameters) *BaseAPIError {
	return &BaseAPIError{
		Type:              BadRequestType,
		Status:            http.StatusBadRequest,
		Title:             BadRequestTitle,
		Instance:          instance(ctx),
		InvalidParameters: invalidFields,
		UnderlyingError:   err,
		Detail:            fmt.Sprintf("%s: %s", BadRequestTitle, invalidFields.String()),
		ctx:               ctx,
	}
}

// NewPreconditionFailedError generates an precondition failed error.
func NewPreconditionFailedError(ctx context.Context, precondition string) *BaseAPIError {
	return &BaseAPIError{
		Type:            PreconditionFailedType,
		Status:          http.StatusPreconditionFailed,
		Title:           PreconditionFailedTitle,
		Instance:        instance(ctx),
		Detail:          MakeSentenceCase(precondition),
		UnderlyingError: fmt.Errorf("precondition failed: %s", precondition),
		ctx:             ctx,
	}
}

// NewRateLimitError generates an HTTP 429 Too Many Requests error.
func NewRateLimitError(ctx context.Context) *BaseAPIError {
	return &BaseAPIError{
		Type:     RateLimitType,
		Status:   http.StatusTooManyRequests,
		Title:    RateLimitDetail,
		Instance: instance(ctx),
		Detail:   RateLimitDetail,
		ctx:      ctx,
	}
}

func NewConflictError(ctx context.Context, err error, detail string) *BaseAPIError {
	return &BaseAPIError{
		Type:            ConflictType,
		Status:          http.StatusConflict,
		Title:           ConflictTitle,
		Instance:        instance(ctx),
		Detail:          detail,
		UnderlyingError: err,
		ctx:             ctx,
	}
}

func NewEmptySetResponse(ctx context.Context, cursorPagination bool) *BaseAPIError {
	bae := &BaseAPIError{
		Status: http.StatusOK,
		ctx:    ctx,
	}

	if cursorPagination {
		bae.Type = EmptySetCursorType
	} else {
		bae.Type = EmptySetType
	}

	return bae
}

func NewNotImplementedError(ctx context.Context, err error) *BaseAPIError {
	return &BaseAPIError{
		Type:   NotImplementedType,
		Status: http.StatusNotImplemented,
		Title:  NotImplementedTitle,
		Detail: NotImplementedDetail,
		ctx:    ctx,
	}
}

// MakeSentenceCase takes any string and returns a Sentence case version of it
func MakeSentenceCase(msg string) string {
	return strings.ToUpper(msg[:1]) + msg[1:]
}

// instance returns the request ID from the context
// TODO: use trace ID from context instead
func instance(ctx context.Context) string {
	reqID := middleware.GetReqID(ctx)
	if reqID != "" {
		return fmt.Sprintf("urn:request:%s", reqID)
	}
	return ""
}
