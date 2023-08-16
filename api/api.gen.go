// Package api provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.13.4 DO NOT EDIT.
package api

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/deepmap/oapi-codegen/pkg/runtime"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Error defines model for Error.
type Error = ErrResponse

// Event CloudEvents Specification JSON Schema
type Event = event.Event

// Meter defines model for Meter.
type Meter = models.Meter

// MeterValue defines model for MeterValue.
type MeterValue = models.MeterValue

// WindowSize defines model for WindowSize.
type WindowSize = models.WindowSize

// IngestEventsApplicationCloudeventsBatchPlusJSONBody defines parameters for IngestEvents.
type IngestEventsApplicationCloudeventsBatchPlusJSONBody = []Event

// GetValuesByMeterIdParams defines parameters for GetValuesByMeterId.
type GetValuesByMeterIdParams struct {
	Subject *string `form:"subject,omitempty" json:"subject,omitempty"`

	// From Start date-time in RFC 3339 format.
	// Must be aligned with the window size.
	// Inclusive.
	From *time.Time `form:"from,omitempty" json:"from,omitempty"`

	// To End date-time in RFC 3339 format.
	// Must be aligned with the window size.
	// Inclusive.
	To *time.Time `form:"to,omitempty" json:"to,omitempty"`

	// WindowSize If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.
	WindowSize *WindowSize `form:"windowSize,omitempty" json:"windowSize,omitempty"`
}

// IngestEventsApplicationCloudeventsPlusJSONRequestBody defines body for IngestEvents for application/cloudevents+json ContentType.
type IngestEventsApplicationCloudeventsPlusJSONRequestBody = Event

// IngestEventsApplicationCloudeventsBatchPlusJSONRequestBody defines body for IngestEvents for application/cloudevents-batch+json ContentType.
type IngestEventsApplicationCloudeventsBatchPlusJSONRequestBody = IngestEventsApplicationCloudeventsBatchPlusJSONBody

// ServerInterface represents all server handlers.
type ServerInterface interface {

	// (POST /api/v1alpha1/events)
	IngestEvents(w http.ResponseWriter, r *http.Request)

	// (GET /api/v1alpha1/meters)
	GetMeters(w http.ResponseWriter, r *http.Request)

	// (GET /api/v1alpha1/meters/{meterId})
	GetMetersById(w http.ResponseWriter, r *http.Request, meterId string)

	// (GET /api/v1alpha1/meters/{meterId}/values)
	GetValuesByMeterId(w http.ResponseWriter, r *http.Request, meterId string, params GetValuesByMeterIdParams)
}

// Unimplemented server implementation that returns http.StatusNotImplemented for each endpoint.

type Unimplemented struct{}

// (POST /api/v1alpha1/events)
func (_ Unimplemented) IngestEvents(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1alpha1/meters)
func (_ Unimplemented) GetMeters(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1alpha1/meters/{meterId})
func (_ Unimplemented) GetMetersById(w http.ResponseWriter, r *http.Request, meterId string) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1alpha1/meters/{meterId}/values)
func (_ Unimplemented) GetValuesByMeterId(w http.ResponseWriter, r *http.Request, meterId string, params GetValuesByMeterIdParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandlerFunc   func(w http.ResponseWriter, r *http.Request, err error)
}

type MiddlewareFunc func(http.Handler) http.Handler

// IngestEvents operation middleware
func (siw *ServerInterfaceWrapper) IngestEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.IngestEvents(w, r)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// GetMeters operation middleware
func (siw *ServerInterfaceWrapper) GetMeters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetMeters(w, r)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// GetMetersById operation middleware
func (siw *ServerInterfaceWrapper) GetMetersById(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "meterId" -------------
	var meterId string

	err = runtime.BindStyledParameterWithLocation("simple", false, "meterId", runtime.ParamLocationPath, chi.URLParam(r, "meterId"), &meterId)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "meterId", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetMetersById(w, r, meterId)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// GetValuesByMeterId operation middleware
func (siw *ServerInterfaceWrapper) GetValuesByMeterId(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "meterId" -------------
	var meterId string

	err = runtime.BindStyledParameterWithLocation("simple", false, "meterId", runtime.ParamLocationPath, chi.URLParam(r, "meterId"), &meterId)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "meterId", Err: err})
		return
	}

	// Parameter object where we will unmarshal all parameters from the context
	var params GetValuesByMeterIdParams

	// ------------- Optional query parameter "subject" -------------

	err = runtime.BindQueryParameter("form", true, false, "subject", r.URL.Query(), &params.Subject)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "subject", Err: err})
		return
	}

	// ------------- Optional query parameter "from" -------------

	err = runtime.BindQueryParameter("form", true, false, "from", r.URL.Query(), &params.From)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "from", Err: err})
		return
	}

	// ------------- Optional query parameter "to" -------------

	err = runtime.BindQueryParameter("form", true, false, "to", r.URL.Query(), &params.To)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "to", Err: err})
		return
	}

	// ------------- Optional query parameter "windowSize" -------------

	err = runtime.BindQueryParameter("form", true, false, "windowSize", r.URL.Query(), &params.WindowSize)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "windowSize", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetValuesByMeterId(w, r, meterId, params)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

type UnescapedCookieParamError struct {
	ParamName string
	Err       error
}

func (e *UnescapedCookieParamError) Error() string {
	return fmt.Sprintf("error unescaping cookie parameter '%s'", e.ParamName)
}

func (e *UnescapedCookieParamError) Unwrap() error {
	return e.Err
}

type UnmarshalingParamError struct {
	ParamName string
	Err       error
}

func (e *UnmarshalingParamError) Error() string {
	return fmt.Sprintf("Error unmarshaling parameter %s as JSON: %s", e.ParamName, e.Err.Error())
}

func (e *UnmarshalingParamError) Unwrap() error {
	return e.Err
}

type RequiredParamError struct {
	ParamName string
}

func (e *RequiredParamError) Error() string {
	return fmt.Sprintf("Query argument %s is required, but not found", e.ParamName)
}

type RequiredHeaderError struct {
	ParamName string
	Err       error
}

func (e *RequiredHeaderError) Error() string {
	return fmt.Sprintf("Header parameter %s is required, but not found", e.ParamName)
}

func (e *RequiredHeaderError) Unwrap() error {
	return e.Err
}

type InvalidParamFormatError struct {
	ParamName string
	Err       error
}

func (e *InvalidParamFormatError) Error() string {
	return fmt.Sprintf("Invalid format for parameter %s: %s", e.ParamName, e.Err.Error())
}

func (e *InvalidParamFormatError) Unwrap() error {
	return e.Err
}

type TooManyValuesForParamError struct {
	ParamName string
	Count     int
}

func (e *TooManyValuesForParamError) Error() string {
	return fmt.Sprintf("Expected one value for %s, got %d", e.ParamName, e.Count)
}

// Handler creates http.Handler with routing matching OpenAPI spec.
func Handler(si ServerInterface) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{})
}

type ChiServerOptions struct {
	BaseURL          string
	BaseRouter       chi.Router
	Middlewares      []MiddlewareFunc
	ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)
}

// HandlerFromMux creates http.Handler with routing matching OpenAPI spec based on the provided mux.
func HandlerFromMux(si ServerInterface, r chi.Router) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{
		BaseRouter: r,
	})
}

func HandlerFromMuxWithBaseURL(si ServerInterface, r chi.Router, baseURL string) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{
		BaseURL:    baseURL,
		BaseRouter: r,
	})
}

// HandlerWithOptions creates http.Handler with additional options
func HandlerWithOptions(si ServerInterface, options ChiServerOptions) http.Handler {
	r := options.BaseRouter

	if r == nil {
		r = chi.NewRouter()
	}
	if options.ErrorHandlerFunc == nil {
		options.ErrorHandlerFunc = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandlerFunc:   options.ErrorHandlerFunc,
	}

	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/api/v1alpha1/events", wrapper.IngestEvents)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1alpha1/meters", wrapper.GetMeters)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1alpha1/meters/{meterId}", wrapper.GetMetersById)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1alpha1/meters/{meterId}/values", wrapper.GetValuesByMeterId)
	})

	return r
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
	"1xkWz95FxXdaDdfrPwMAAP//lDgshnYYAAA=",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
