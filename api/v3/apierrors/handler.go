package apierrors

import (
	"context"
	"errors"
	"net/http"
	"strings"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/models"
)

const httpStatusCodeErrorAttribute = "openmeter.http.status_code"

// NewV3ErrorHandlerFunc returns an oapi-codegen ChiServerOptions.ErrorHandlerFunc implementation.
//
// It is invoked when the generated router fails request binding (query/path/header parsing).
// The main purpose is to ensure we always write a response (otherwise net/http defaults to 200 with
// an empty body), and to keep error-to-status mapping consistent with our model error types.
func NewV3ErrorHandlerFunc(logger errorsx.Handler) func(w http.ResponseWriter, r *http.Request, err error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		if err == nil {
			return
		}

		// If it's already a v3 API error, just render it.
		var apiErr *BaseAPIError
		if errors.As(err, &apiErr) {
			apiErr.HandleAPIError(w, r)
			return
		}

		ctx := r.Context()

		// Request binding errors produced by the generated v3 router.
		// Convert them into v3 InvalidParameters so the response is actionable for clients.
		if invalidParams, ok := invalidParametersFromGeneratedRouterError(err); ok {
			logger.HandleContext(ctx, err)
			NewBadRequestError(ctx, err, invalidParams).HandleAPIError(w, r)
			return
		}

		// Mirror commonhttp.GenericErrorEncoder's ordering, but render using v3 apierrors.
		if status, ok := singularHTTPStatusFromValidationIssues(err); ok {
			if mapped := apiErrorFromHTTPStatus(ctx, status, err); mapped != nil {
				logger.HandleContext(ctx, err)
				mapped.HandleAPIError(w, r)
				return
			}
		}

		// Default: classify as validation error (400) for request binding failures.
		validationErr := models.NewGenericValidationError(err)
		logger.HandleContext(r.Context(), validationErr)
		NewBadRequestError(r.Context(), validationErr, nil).HandleAPIError(w, r)
	}
}

func invalidParametersFromGeneratedRouterError(err error) (InvalidParameters, bool) {
	// These types are defined in api/v3/api.gen.go.
	//
	// Note: those errors do not carry the parameter location (query/path/header) except for the
	// dedicated "required header" variant, so we default to "query" where ambiguous. This is still
	// a major improvement over returning an empty error response.
	var invalidFormat *api.InvalidParamFormatError
	if errors.As(err, &invalidFormat) {
		field := enrichFieldFromBindError(invalidFormat.ParamName, invalidFormat.Err.Error())
		return InvalidParameters{
			{
				Field:  field,
				Rule:   "format",
				Reason: invalidFormat.Err.Error(),
				Source: InvalidParamSourceQuery,
			},
		}, true
	}

	var requiredParam *api.RequiredParamError
	if errors.As(err, &requiredParam) {
		return InvalidParameters{
			{
				Field:  requiredParam.ParamName,
				Rule:   "required",
				Reason: "is required",
				Source: InvalidParamSourceQuery,
			},
		}, true
	}

	var requiredHeader *api.RequiredHeaderError
	if errors.As(err, &requiredHeader) {
		return InvalidParameters{
			{
				Field:  requiredHeader.ParamName,
				Rule:   "required",
				Reason: "is required",
				Source: InvalidParamSourceHeader,
			},
		}, true
	}

	var tooMany *api.TooManyValuesForParamError
	if errors.As(err, &tooMany) {
		return InvalidParameters{
			{
				Field:  tooMany.ParamName,
				Rule:   "too_many_values",
				Reason: tooMany.Error(),
				Source: InvalidParamSourceQuery,
			},
		}, true
	}

	var unmarshal *api.UnmarshalingParamError
	if errors.As(err, &unmarshal) {
		return InvalidParameters{
			{
				Field:  unmarshal.ParamName,
				Rule:   "unmarshal",
				Reason: unmarshal.Err.Error(),
				Source: InvalidParamSourceQuery,
			},
		}, true
	}

	var unescapedCookie *api.UnescapedCookieParamError
	if errors.As(err, &unescapedCookie) {
		return InvalidParameters{
			{
				Field:  unescapedCookie.ParamName,
				Rule:   "unescape",
				Reason: unescapedCookie.Error(),
				Source: InvalidParamSourceHeader,
			},
		}, true
	}

	return nil, false
}

func enrichFieldFromBindError(paramName string, bindErrMsg string) string {
	// oapi-codegen deepObject binding errors (runtime.BindQueryParameter) can be more specific than
	// just the outer parameter name, e.g.:
	// "error assigning value to destination: field [sizee] is not present in destination object".
	//
	// For nicer AIP errors, return "page.sizee" instead of just "page".
	if paramName == "" || bindErrMsg == "" {
		return paramName
	}
	if strings.Contains(paramName, "[") {
		// Already specific (e.g. "page[size]") - keep as-is.
		return paramName
	}
	const needle = "field ["
	i := strings.Index(bindErrMsg, needle)
	if i == -1 {
		return paramName
	}
	rest := bindErrMsg[i+len(needle):]
	j := strings.Index(rest, "]")
	if j == -1 {
		return paramName
	}
	field := rest[:j]
	if field == "" {
		return paramName
	}
	return paramName + "." + field
}

func singularHTTPStatusFromValidationIssues(err error) (int, bool) {
	issues, _ := models.AsValidationIssues(err)
	if len(issues) == 0 {
		return 0, false
	}

	// We intentionally mirror commonhttp.HandleIssueIfHTTPStatusKnown's "singular" behavior:
	// if multiple status codes are present, we don't map.
	codes := make(map[int]struct{}, 1)
	for _, issue := range issues {
		raw, ok := issue.Attributes()[httpStatusCodeErrorAttribute]
		if !ok {
			continue
		}
		c, ok := raw.(int)
		if !ok {
			continue
		}
		codes[c] = struct{}{}
	}

	if len(codes) != 1 {
		return 0, false
	}

	for c := range codes {
		return c, true
	}
	return 0, false
}

func apiErrorFromHTTPStatus(ctx context.Context, status int, err error) *BaseAPIError {
	switch status {
	case http.StatusBadRequest:
		return NewBadRequestError(ctx, err, nil)
	case http.StatusUnauthorized:
		return NewUnauthenticatedError(ctx, err)
	case http.StatusForbidden:
		return NewForbiddenError(ctx, err)
	case http.StatusNotFound:
		return NewNotFoundError(ctx, err, "")
	case http.StatusConflict:
		return NewConflictError(ctx, err, err.Error())
	case http.StatusPreconditionFailed:
		return NewPreconditionFailedError(ctx, err.Error())
	case http.StatusNotImplemented:
		return NewNotImplementedError(ctx, err)
	default:
		return nil
	}
}
