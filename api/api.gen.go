// Package api provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version (devel) DO NOT EDIT.
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

// Event CloudEvents Specification JSON Schema
type Event = event.Event

// IdOrSlug defines model for IdOrSlug.
type IdOrSlug = string

// Meter defines model for Meter.
type Meter = models.Meter

// MeterAggregation The aggregation type to use for the meter.
type MeterAggregation = models.MeterAggregation

// MeterValue defines model for MeterValue.
type MeterValue = models.MeterValue

// NS defines model for Namespace.
type NS = models.Namespace

// Problem A Problem Details object (RFC 7807)
type Problem = models.Problem

// WindowSize defines model for WindowSize.
type WindowSize = models.WindowSize

// MeterIdOrSlug defines model for meterIdOrSlug.
type MeterIdOrSlug = IdOrSlug

// Namespace defines model for namespace.
type Namespace = string

// IngestEventsParams defines parameters for IngestEvents.
type IngestEventsParams struct {
	// OMNamespace Optional namespace
	Namespace *Namespace `json:"OM-Namespace,omitempty"`
}

// ListMetersParams defines parameters for ListMeters.
type ListMetersParams struct {
	// OMNamespace Optional namespace
	Namespace *Namespace `json:"OM-Namespace,omitempty"`
}

// CreateMeterParams defines parameters for CreateMeter.
type CreateMeterParams struct {
	// OMNamespace Optional namespace
	Namespace *Namespace `json:"OM-Namespace,omitempty"`
}

// DeleteMeterParams defines parameters for DeleteMeter.
type DeleteMeterParams struct {
	// OMNamespace Optional namespace
	Namespace *Namespace `json:"OM-Namespace,omitempty"`
}

// GetMeterParams defines parameters for GetMeter.
type GetMeterParams struct {
	// OMNamespace Optional namespace
	Namespace *Namespace `json:"OM-Namespace,omitempty"`
}

// GetMeterValuesParams defines parameters for GetMeterValues.
type GetMeterValuesParams struct {
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

	// OMNamespace Optional namespace
	Namespace *Namespace `json:"OM-Namespace,omitempty"`
}

// IngestEventsJSONRequestBody defines body for IngestEvents for application/cloudevents+json ContentType.
type IngestEventsJSONRequestBody = Event

// CreateMeterJSONRequestBody defines body for CreateMeter for application/json ContentType.
type CreateMeterJSONRequestBody = Meter

// CreateNamespaceJSONRequestBody defines body for CreateNamespace for application/json ContentType.
type CreateNamespaceJSONRequestBody = NS

// ServerInterface represents all server handlers.
type ServerInterface interface {

	// (POST /api/v1alpha2/events)
	IngestEvents(w http.ResponseWriter, r *http.Request, params IngestEventsParams)

	// (GET /api/v1alpha2/meters)
	ListMeters(w http.ResponseWriter, r *http.Request, params ListMetersParams)

	// (POST /api/v1alpha2/meters)
	CreateMeter(w http.ResponseWriter, r *http.Request, params CreateMeterParams)

	// (DELETE /api/v1alpha2/meters/{meterIdOrSlug})
	DeleteMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug MeterIdOrSlug, params DeleteMeterParams)

	// (GET /api/v1alpha2/meters/{meterIdOrSlug})
	GetMeter(w http.ResponseWriter, r *http.Request, meterIdOrSlug MeterIdOrSlug, params GetMeterParams)

	// (GET /api/v1alpha2/meters/{meterIdOrSlug}/values)
	GetMeterValues(w http.ResponseWriter, r *http.Request, meterIdOrSlug MeterIdOrSlug, params GetMeterValuesParams)

	// (POST /api/v1alpha2/namespaces)
	CreateNamespace(w http.ResponseWriter, r *http.Request)
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

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params IngestEventsParams

	headers := r.Header

	// ------------- Optional header parameter "OM-Namespace" -------------
	if valueList, found := headers[http.CanonicalHeaderKey("OM-Namespace")]; found {
		var Namespace Namespace
		n := len(valueList)
		if n != 1 {
			siw.ErrorHandlerFunc(w, r, &TooManyValuesForParamError{ParamName: "OM-Namespace", Count: n})
			return
		}

		err = runtime.BindStyledParameterWithLocation("simple", false, "OM-Namespace", runtime.ParamLocationHeader, valueList[0], &Namespace)
		if err != nil {
			siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "OM-Namespace", Err: err})
			return
		}

		params.Namespace = &Namespace

	}

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.IngestEvents(w, r, params)
	})

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// ListMeters operation middleware
func (siw *ServerInterfaceWrapper) ListMeters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params ListMetersParams

	headers := r.Header

	// ------------- Optional header parameter "OM-Namespace" -------------
	if valueList, found := headers[http.CanonicalHeaderKey("OM-Namespace")]; found {
		var Namespace Namespace
		n := len(valueList)
		if n != 1 {
			siw.ErrorHandlerFunc(w, r, &TooManyValuesForParamError{ParamName: "OM-Namespace", Count: n})
			return
		}

		err = runtime.BindStyledParameterWithLocation("simple", false, "OM-Namespace", runtime.ParamLocationHeader, valueList[0], &Namespace)
		if err != nil {
			siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "OM-Namespace", Err: err})
			return
		}

		params.Namespace = &Namespace

	}

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.ListMeters(w, r, params)
	})

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// CreateMeter operation middleware
func (siw *ServerInterfaceWrapper) CreateMeter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params CreateMeterParams

	headers := r.Header

	// ------------- Optional header parameter "OM-Namespace" -------------
	if valueList, found := headers[http.CanonicalHeaderKey("OM-Namespace")]; found {
		var Namespace Namespace
		n := len(valueList)
		if n != 1 {
			siw.ErrorHandlerFunc(w, r, &TooManyValuesForParamError{ParamName: "OM-Namespace", Count: n})
			return
		}

		err = runtime.BindStyledParameterWithLocation("simple", false, "OM-Namespace", runtime.ParamLocationHeader, valueList[0], &Namespace)
		if err != nil {
			siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "OM-Namespace", Err: err})
			return
		}

		params.Namespace = &Namespace

	}

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.CreateMeter(w, r, params)
	})

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

	// Parameter object where we will unmarshal all parameters from the context
	var params DeleteMeterParams

	headers := r.Header

	// ------------- Optional header parameter "OM-Namespace" -------------
	if valueList, found := headers[http.CanonicalHeaderKey("OM-Namespace")]; found {
		var Namespace Namespace
		n := len(valueList)
		if n != 1 {
			siw.ErrorHandlerFunc(w, r, &TooManyValuesForParamError{ParamName: "OM-Namespace", Count: n})
			return
		}

		err = runtime.BindStyledParameterWithLocation("simple", false, "OM-Namespace", runtime.ParamLocationHeader, valueList[0], &Namespace)
		if err != nil {
			siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "OM-Namespace", Err: err})
			return
		}

		params.Namespace = &Namespace

	}

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.DeleteMeter(w, r, meterIdOrSlug, params)
	})

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

	// Parameter object where we will unmarshal all parameters from the context
	var params GetMeterParams

	headers := r.Header

	// ------------- Optional header parameter "OM-Namespace" -------------
	if valueList, found := headers[http.CanonicalHeaderKey("OM-Namespace")]; found {
		var Namespace Namespace
		n := len(valueList)
		if n != 1 {
			siw.ErrorHandlerFunc(w, r, &TooManyValuesForParamError{ParamName: "OM-Namespace", Count: n})
			return
		}

		err = runtime.BindStyledParameterWithLocation("simple", false, "OM-Namespace", runtime.ParamLocationHeader, valueList[0], &Namespace)
		if err != nil {
			siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "OM-Namespace", Err: err})
			return
		}

		params.Namespace = &Namespace

	}

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetMeter(w, r, meterIdOrSlug, params)
	})

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// GetMeterValues operation middleware
func (siw *ServerInterfaceWrapper) GetMeterValues(w http.ResponseWriter, r *http.Request) {
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
	var params GetMeterValuesParams

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

	headers := r.Header

	// ------------- Optional header parameter "OM-Namespace" -------------
	if valueList, found := headers[http.CanonicalHeaderKey("OM-Namespace")]; found {
		var Namespace Namespace
		n := len(valueList)
		if n != 1 {
			siw.ErrorHandlerFunc(w, r, &TooManyValuesForParamError{ParamName: "OM-Namespace", Count: n})
			return
		}

		err = runtime.BindStyledParameterWithLocation("simple", false, "OM-Namespace", runtime.ParamLocationHeader, valueList[0], &Namespace)
		if err != nil {
			siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "OM-Namespace", Err: err})
			return
		}

		params.Namespace = &Namespace

	}

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetMeterValues(w, r, meterIdOrSlug, params)
	})

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// CreateNamespace operation middleware
func (siw *ServerInterfaceWrapper) CreateNamespace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.CreateNamespace(w, r)
	})

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

type UnmarshallingParamError struct {
	ParamName string
	Err       error
}

func (e *UnmarshallingParamError) Error() string {
	return fmt.Sprintf("Error unmarshalling parameter %s as JSON: %s", e.ParamName, e.Err.Error())
}

func (e *UnmarshallingParamError) Unwrap() error {
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
		r.Post(options.BaseURL+"/api/v1alpha2/events", wrapper.IngestEvents)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1alpha2/meters", wrapper.ListMeters)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/api/v1alpha2/meters", wrapper.CreateMeter)
	})
	r.Group(func(r chi.Router) {
		r.Delete(options.BaseURL+"/api/v1alpha2/meters/{meterIdOrSlug}", wrapper.DeleteMeter)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1alpha2/meters/{meterIdOrSlug}", wrapper.GetMeter)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1alpha2/meters/{meterIdOrSlug}/values", wrapper.GetMeterValues)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/api/v1alpha2/namespaces", wrapper.CreateNamespace)
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
		return nil, fmt.Errorf("error base64 decoding spec: %s", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
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
	var res = make(map[string]func() ([]byte, error))
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
	var resolvePath = PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		var pathToFile = url.String()
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
