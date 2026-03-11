package oasmiddleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	validatorerrors "github.com/pb33f/libopenapi-validator/errors"
	"gopkg.in/yaml.v3"
)

type (
	RequestNotFoundHookFunc     = func(error, http.ResponseWriter, *http.Request) bool
	RequestValidationErrorFunc  = func(error, http.ResponseWriter, *http.Request) bool
	ResponseValidationFunc     = func(error, *http.Request)
)

// ValidateRequestOption provides the hook functions for the validation middleware.
type ValidateRequestOption struct {
	// RouteNotFoundHook is called when the route is not found at the spec level.
	// If the hook returns `true` the request flow is stopped.
	RouteNotFoundHook RequestNotFoundHookFunc
	// RouteValidationErrorHook is called when the route parameters or body are
	// not validated. If the hook returns `true` the request flow is stopped.
	RouteValidationErrorHook RequestValidationErrorFunc
}

// ValidateResponseOption provides the hook function for response validation.
type ValidateResponseOption struct {
	// ResponseValidationErrorHook is called when the route response body is not validated.
	ResponseValidationErrorHook ResponseValidationFunc
}

// NewValidator creates a libopenapi-validator from the given spec bytes and base URL.
// The baseURL is set as the server URL for path matching (e.g. /api/v3).
func NewValidator(specBytes []byte, baseURL string) (validator.Validator, error) {
	patched, err := patchSpecServers(specBytes, baseURL)
	if err != nil {
		return nil, err
	}

	document, err := libopenapi.NewDocument(patched)
	if err != nil {
		return nil, err
	}

	v, errs := validator.NewValidator(document)
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return v, nil
}

// patchSpecServers modifies the OpenAPI spec to set the servers URL for path matching.
func patchSpecServers(specBytes []byte, baseURL string) ([]byte, error) {
	var spec map[string]any
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		if err := yaml.Unmarshal(specBytes, &spec); err != nil {
			return nil, fmt.Errorf("parse spec: %w", err)
		}
	}

	spec["servers"] = []map[string]any{
		{"url": baseURL},
	}

	return json.Marshal(spec)
}

// filterQueryParamErrors removes validation errors for multi-level nested deepObject
// query params. libopenapi-validator has a known issue: it works for objects with
// depth of one but fails for nested objects (e.g. filter[provider][eq]=x).
// See https://github.com/pb33f/libopenapi-validator/issues/83
func filterQueryParamErrors(errs []*validatorerrors.ValidationError) []*validatorerrors.ValidationError {
	if len(errs) == 0 {
		return errs
	}
	var filtered []*validatorerrors.ValidationError
	for _, e := range errs {
		if e == nil {
			continue
		}
		// Only skip errors for object-type query params that failed schema validation
		// (e.g. "The query parameter 'filter' is defined as an object, however it
		// failed to pass a schema validation"). Simple query params are still validated.
		msg := strings.ToLower(e.Message)
		reason := strings.ToLower(e.Reason)
		isQueryParam := strings.Contains(msg, "query parameter") || strings.Contains(reason, "query parameter")
		isObjectSchema := strings.Contains(msg, "object") && strings.Contains(msg, "schema validation") ||
			strings.Contains(reason, "object") && strings.Contains(reason, "schema validation")
		if isQueryParam && isObjectSchema {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

// isRouteNotFound returns true if any validation error indicates path or operation not found.
func isRouteNotFound(errs []*validatorerrors.ValidationError) bool {
	for _, e := range errs {
		if e != nil && (e.IsPathMissingError() || e.IsOperationMissingError()) {
			return true
		}
	}
	return false
}

// LibopenapiValidationErrors wraps libopenapi validation errors for use in hooks.
type LibopenapiValidationErrors struct {
	Errors []*validatorerrors.ValidationError
}

func (e *LibopenapiValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	return e.Errors[0].Error()
}

// ValidateRequest is the middleware to validate the request against the OpenAPI spec.
// Validation errors for multi-level nested deepObject query params are filtered out
// due to libopenapi-validator issue #83 (works for depth 1, fails for nested objects).
func ValidateRequest(v validator.Validator, opts ValidateRequestOption) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			skipServe := false
			valid, validationErrors := v.ValidateHttpRequest(r)
			if !valid {
				// Filter out multi-level nested deepObject query param errors (issue #83).
				validationErrors = filterQueryParamErrors(validationErrors)
				if len(validationErrors) == 0 {
					valid = true
				}
			}
			if !valid {
				err := &LibopenapiValidationErrors{Errors: validationErrors}
				if opts.RouteNotFoundHook != nil && isRouteNotFound(validationErrors) {
					skipServe = opts.RouteNotFoundHook(err, w, r)
				} else if opts.RouteValidationErrorHook != nil {
					skipServe = opts.RouteValidationErrorHook(err, w, r)
				}
			}
			if !skipServe {
				h.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(fn)
	}
}

// ValidateResponse is the middleware to validate the response against the OpenAPI spec.
func ValidateResponse(v validator.Validator, opts ValidateResponseOption) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			rww := NewResponseWriterWrapper(w)
			h.ServeHTTP(rww, r)

			b := new(bytes.Buffer)
			if _, err := b.ReadFrom(rww.Body()); err != nil {
				return
			}

			resp := &http.Response{
				StatusCode: *rww.StatusCode(),
				Header:     rww.Header(),
				Body:       io.NopCloser(bytes.NewReader(b.Bytes())),
			}

			valid, validationErrors := v.ValidateHttpResponse(r, resp)
			if !valid && opts.ResponseValidationErrorHook != nil {
				opts.ResponseValidationErrorHook(&LibopenapiValidationErrors{Errors: validationErrors}, r)
			}
		}
		return http.HandlerFunc(fn)
	}
}
