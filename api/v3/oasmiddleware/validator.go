package oasmiddleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
)

type (
	RequestNotFoundHookFunc    = func(error, http.ResponseWriter, *http.Request) bool
	RequestValidationErrorFunc = func(error, http.ResponseWriter, *http.Request) bool
	ResponseValidationFunc     = func(error, *http.Request)
)

// ValidateRequestOption provides the hook functions and the openapi3filter
// option to be passed in to the underlying library
type ValidateRequestOption struct {
	// RouteNotFoundHook is called when the route is not found at the spec level
	// if the hook returns `true` the request flow is stopped
	RouteNotFoundHook RequestNotFoundHookFunc
	// RouteValidationErrorHook is called when the route parameters or body are
	// not validated. if the hook returns `true` the request flow is stopped
	RouteValidationErrorHook RequestValidationErrorFunc
	// FilterOptions are the openapi3filter option to pass to the underlying lib
	FilterOptions *openapi3filter.Options
}

// ValidateRequest is the middleware to be used to validate the request to the spec
// passed in for the validation router
func ValidateRequest(validationRouter routers.Router, opts ValidateRequestOption) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			skipServe := false
			route, pathParams, err := validationRouter.FindRoute(r.WithContext(ctx))
			if err != nil {
				if opts.RouteNotFoundHook != nil {
					skipServe = opts.RouteNotFoundHook(err, w, r)
				}
			} else {
				requestValidationInput := &openapi3filter.RequestValidationInput{
					Request:    r,
					PathParams: pathParams,
					Route:      route,
					Options:    opts.FilterOptions,
				}
				if err := openapi3filter.ValidateRequest(ctx, requestValidationInput); err != nil {
					if opts.RouteValidationErrorHook != nil {
						skipServe = opts.RouteValidationErrorHook(err, w, r)
					}
				}
			}
			if !skipServe {
				h.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(fn)
	}
}

// ValidateResponseOption provides the hook function and the openapi3filter
// option to be passed in to the underlying library
type ValidateResponseOption struct {
	// ResponseValidationErrorHook is called when the route response body is not validated
	ResponseValidationErrorHook ResponseValidationFunc
	// FilterOptions are the openapi3filter option to pass to the underlying lib
	FilterOptions *openapi3filter.Options
}

// ValidateResponse is the middleware to be used to validate the response to the spec
// passed in for the validation router
func ValidateResponse(validationRouter routers.Router, opts ValidateResponseOption) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			var err error

			route, pathParams, err := validationRouter.FindRoute(r)

			if err != nil {
				h.ServeHTTP(w, r)
				if opts.ResponseValidationErrorHook != nil {
					opts.ResponseValidationErrorHook(err, r)
				}
			} else {
				// need to wrap std lib response to access the body
				rww := NewResponseWriterWrapper(w)

				h.ServeHTTP(rww, r)

				b := new(bytes.Buffer)
				_, err := b.ReadFrom(rww.Body())
				if err != nil {
					return
				}
				bodyReader := bytes.NewReader(b.Bytes())

				responseValidationInput := &openapi3filter.ResponseValidationInput{
					RequestValidationInput: &openapi3filter.RequestValidationInput{
						Request:    r,
						PathParams: pathParams,
						Route:      route,
						Options:    opts.FilterOptions,
					},
					Header: rww.Header(),
					Body:   io.NopCloser(bodyReader),
					Status: *rww.StatusCode(),
				}

				if err := openapi3filter.ValidateResponse(r.Context(), responseValidationInput); err != nil {
					if opts.ResponseValidationErrorHook != nil {
						opts.ResponseValidationErrorHook(err, r)
					}
				}
			}
		}
		return http.HandlerFunc(fn)
	}
}
