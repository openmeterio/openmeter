// Package api provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen/v2 version v2.0.0 DO NOT EDIT.
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
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/oapi-codegen/runtime"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Event CloudEvents Specification JSON Schema
type Event = event.Event

// IdOrSlug defines model for IdOrSlug.
type IdOrSlug = string

// Meter defines model for Meter.
type Meter = models.Meter

// MeterAggregation The aggregation type to use for the meter.
type MeterAggregation = models.MeterAggregation

// MeterQueryRow defines model for MeterQueryRow.
type MeterQueryRow = models.MeterQueryRow

// Problem A Problem Details object (RFC 7807)
type Problem = models.StatusProblem

// WindowSize defines model for WindowSize.
type WindowSize = models.WindowSize

// MeterIdOrSlug defines model for meterIdOrSlug.
type MeterIdOrSlug = IdOrSlug

// BadRequestProblemResponse A Problem Details object (RFC 7807)
type BadRequestProblemResponse = Problem

// NotFoundProblemResponse A Problem Details object (RFC 7807)
type NotFoundProblemResponse = Problem

// NotImplementedProblemResponse A Problem Details object (RFC 7807)
type NotImplementedProblemResponse = Problem

// UnexpectedProblemResponse A Problem Details object (RFC 7807)
type UnexpectedProblemResponse = Problem

// ListEventsParams defines parameters for ListEvents.
type ListEventsParams struct {
	// Limit Number of events to return.
	Limit *int `form:"limit,omitempty" json:"limit,omitempty"`
}

// IngestEventsApplicationCloudeventsBatchPlusJSONBody defines parameters for IngestEvents.
type IngestEventsApplicationCloudeventsBatchPlusJSONBody = []Event

// QueryMeterParams defines parameters for QueryMeter.
type QueryMeterParams struct {
	// From Start date-time in RFC 3339 format in UTC timezone.
	// Must be aligned with the window size.
	// Inclusive.
	From *time.Time `form:"from,omitempty" json:"from,omitempty"`

	// To End date-time in RFC 3339 format in UTC timezone.
	// Must be aligned with the window size.
	// Inclusive.
	To *time.Time `form:"to,omitempty" json:"to,omitempty"`

	// WindowSize If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.
	WindowSize *WindowSize `form:"windowSize,omitempty" json:"windowSize,omitempty"`

	// Aggregation If not specified, OpenMeter will use the default aggregation type.
	// As OpenMeter stores aggregates defined by meter config, passing a different aggregate can lead to inaccurate results.
	// For example getting the MIN of SUMs.
	Aggregation *MeterAggregation `form:"aggregation,omitempty" json:"aggregation,omitempty"`
	Subject     *[]string         `form:"subject,omitempty" json:"subject,omitempty"`

	// GroupBy If not specified a single aggregate will be returned for each subject and time window.
	GroupBy *[]string `form:"groupBy,omitempty" json:"groupBy,omitempty"`
}

// IngestEventsApplicationCloudeventsPlusJSONRequestBody defines body for IngestEvents for application/cloudevents+json ContentType.
type IngestEventsApplicationCloudeventsPlusJSONRequestBody = Event

// IngestEventsApplicationCloudeventsBatchPlusJSONRequestBody defines body for IngestEvents for application/cloudevents-batch+json ContentType.
type IngestEventsApplicationCloudeventsBatchPlusJSONRequestBody = IngestEventsApplicationCloudeventsBatchPlusJSONBody

// CreateMeterJSONRequestBody defines body for CreateMeter for application/json ContentType.
type CreateMeterJSONRequestBody = Meter

// ServerInterface represents all server handlers.
type ServerInterface interface {

	// (GET /api/v1/events)
	ListEvents(w http.ResponseWriter, r *http.Request, params ListEventsParams)

	// (POST /api/v1/events)
	IngestEvents(w http.ResponseWriter, r *http.Request)

	// (GET /api/v1/meters)
	ListMeters(w http.ResponseWriter, r *http.Request)

	// (POST /api/v1/meters)
	CreateMeter(w http.ResponseWriter, r *http.Request)

	// (DELETE /api/v1/meters/{meterIdOrSlug})
	DeleteMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug MeterIdOrSlug)

	// (GET /api/v1/meters/{meterIdOrSlug})
	GetMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug MeterIdOrSlug)

	// (GET /api/v1/meters/{meterIdOrSlug}/query)
	QueryMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug MeterIdOrSlug, params QueryMeterParams)

	// (GET /api/v1/meters/{meterIdOrSlug}/subjects)
	ListMeterSubjects(w http.ResponseWriter, r *http.Request, meterIdOrSlug MeterIdOrSlug)
}

// Unimplemented server implementation that returns http.StatusNotImplemented for each endpoint.

type Unimplemented struct{}

// (GET /api/v1/events)
func (_ Unimplemented) ListEvents(w http.ResponseWriter, r *http.Request, params ListEventsParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (POST /api/v1/events)
func (_ Unimplemented) IngestEvents(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1/meters)
func (_ Unimplemented) ListMeters(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (POST /api/v1/meters)
func (_ Unimplemented) CreateMeter(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (DELETE /api/v1/meters/{meterIdOrSlug})
func (_ Unimplemented) DeleteMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug MeterIdOrSlug) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1/meters/{meterIdOrSlug})
func (_ Unimplemented) GetMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug MeterIdOrSlug) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1/meters/{meterIdOrSlug}/query)
func (_ Unimplemented) QueryMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug MeterIdOrSlug, params QueryMeterParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1/meters/{meterIdOrSlug}/subjects)
func (_ Unimplemented) ListMeterSubjects(w http.ResponseWriter, r *http.Request, meterIdOrSlug MeterIdOrSlug) {
	w.WriteHeader(http.StatusNotImplemented)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandlerFunc   func(w http.ResponseWriter, r *http.Request, err error)
}

type MiddlewareFunc func(http.Handler) http.Handler

// ListEvents operation middleware
func (siw *ServerInterfaceWrapper) ListEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params ListEventsParams

	// ------------- Optional query parameter "limit" -------------

	err = runtime.BindQueryParameter("form", true, false, "limit", r.URL.Query(), &params.Limit)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "limit", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.ListEvents(w, r, params)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

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

// ListMeters operation middleware
func (siw *ServerInterfaceWrapper) ListMeters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.ListMeters(w, r)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// CreateMeter operation middleware
func (siw *ServerInterfaceWrapper) CreateMeter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.CreateMeter(w, r)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// DeleteMeter operation middleware
func (siw *ServerInterfaceWrapper) DeleteMeter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "meterIdOrSlug" -------------
	var meterIdOrSlug MeterIdOrSlug

	err = runtime.BindStyledParameterWithLocation("simple", false, "meterIdOrSlug", runtime.ParamLocationPath, chi.URLParam(r, "meterIdOrSlug"), &meterIdOrSlug)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "meterIdOrSlug", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.DeleteMeter(w, r, meterIdOrSlug)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// GetMeter operation middleware
func (siw *ServerInterfaceWrapper) GetMeter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "meterIdOrSlug" -------------
	var meterIdOrSlug MeterIdOrSlug

	err = runtime.BindStyledParameterWithLocation("simple", false, "meterIdOrSlug", runtime.ParamLocationPath, chi.URLParam(r, "meterIdOrSlug"), &meterIdOrSlug)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "meterIdOrSlug", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetMeter(w, r, meterIdOrSlug)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// QueryMeter operation middleware
func (siw *ServerInterfaceWrapper) QueryMeter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "meterIdOrSlug" -------------
	var meterIdOrSlug MeterIdOrSlug

	err = runtime.BindStyledParameterWithLocation("simple", false, "meterIdOrSlug", runtime.ParamLocationPath, chi.URLParam(r, "meterIdOrSlug"), &meterIdOrSlug)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "meterIdOrSlug", Err: err})
		return
	}

	// Parameter object where we will unmarshal all parameters from the context
	var params QueryMeterParams

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

	// ------------- Optional query parameter "aggregation" -------------

	err = runtime.BindQueryParameter("form", true, false, "aggregation", r.URL.Query(), &params.Aggregation)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "aggregation", Err: err})
		return
	}

	// ------------- Optional query parameter "subject" -------------

	err = runtime.BindQueryParameter("form", true, false, "subject", r.URL.Query(), &params.Subject)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "subject", Err: err})
		return
	}

	// ------------- Optional query parameter "groupBy" -------------

	err = runtime.BindQueryParameter("form", true, false, "groupBy", r.URL.Query(), &params.GroupBy)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "groupBy", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.QueryMeter(w, r, meterIdOrSlug, params)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// ListMeterSubjects operation middleware
func (siw *ServerInterfaceWrapper) ListMeterSubjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "meterIdOrSlug" -------------
	var meterIdOrSlug MeterIdOrSlug

	err = runtime.BindStyledParameterWithLocation("simple", false, "meterIdOrSlug", runtime.ParamLocationPath, chi.URLParam(r, "meterIdOrSlug"), &meterIdOrSlug)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "meterIdOrSlug", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.ListMeterSubjects(w, r, meterIdOrSlug)
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
		r.Get(options.BaseURL+"/api/v1/events", wrapper.ListEvents)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/api/v1/events", wrapper.IngestEvents)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/meters", wrapper.ListMeters)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/api/v1/meters", wrapper.CreateMeter)
	})
	r.Group(func(r chi.Router) {
		r.Delete(options.BaseURL+"/api/v1/meters/{meterIdOrSlug}", wrapper.DeleteMeter)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/meters/{meterIdOrSlug}", wrapper.GetMeter)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/meters/{meterIdOrSlug}/query", wrapper.QueryMeter)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/meters/{meterIdOrSlug}/subjects", wrapper.ListMeterSubjects)
	})

	return r
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
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
