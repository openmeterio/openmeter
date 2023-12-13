// Package api provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen/v2 version v2.0.0 DO NOT EDIT.
package api

import (
	"bytes"
	"compress/gzip"
	"context"
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

const (
	PortalTokenScopes = "portalToken.Scopes"
)

// Event CloudEvents Specification JSON Schema
type Event = event.Event

// IdOrSlug defines model for IdOrSlug.
type IdOrSlug = string

// IngestedEvent defines model for IngestedEvent.
type IngestedEvent struct {
	// Event CloudEvents Specification JSON Schema
	Event           Event   `json:"event"`
	ValidationError *string `json:"validationError,omitempty"`
}

// Meter defines model for Meter.
type Meter = models.Meter

// MeterAggregation The aggregation type to use for the meter.
type MeterAggregation = models.MeterAggregation

// MeterQueryResult defines model for MeterQueryResult.
type MeterQueryResult struct {
	Data []MeterQueryRow `json:"data"`
	From *time.Time      `json:"from,omitempty"`
	To   *time.Time      `json:"to,omitempty"`

	// WindowSize Aggregation window size.
	WindowSize *WindowSize `json:"windowSize,omitempty"`
}

// MeterQueryRow defines model for MeterQueryRow.
type MeterQueryRow = models.MeterQueryRow

// PortalToken defines model for PortalToken.
type PortalToken struct {
	AllowedMeterSlugs *[]string  `json:"allowedMeterSlugs,omitempty"`
	CreatedAt         *time.Time `json:"createdAt,omitempty"`
	ExpiresAt         *time.Time `json:"expiresAt,omitempty"`
	Id                *string    `json:"id,omitempty"`
	Subject           string     `json:"subject"`
	Token             *string    `json:"token,omitempty"`
}

// Problem A Problem Details object (RFC 7807)
type Problem = models.StatusProblem

// WindowSize Aggregation window size.
type WindowSize = models.WindowSize

// MeterIdOrSlug defines model for meterIdOrSlug.
type MeterIdOrSlug = IdOrSlug

// QueryFrom defines model for queryFrom.
type QueryFrom = time.Time

// QueryGroupBy defines model for queryGroupBy.
type QueryGroupBy = []string

// QuerySubject defines model for querySubject.
type QuerySubject = []string

// QueryTo defines model for queryTo.
type QueryTo = time.Time

// QueryWindowSize Aggregation window size.
type QueryWindowSize = WindowSize

// QueryWindowTimeZone defines model for queryWindowTimeZone.
type QueryWindowTimeZone = string

// BadRequestProblemResponse A Problem Details object (RFC 7807)
type BadRequestProblemResponse = Problem

// NotFoundProblemResponse A Problem Details object (RFC 7807)
type NotFoundProblemResponse = Problem

// NotImplementedProblemResponse A Problem Details object (RFC 7807)
type NotImplementedProblemResponse = Problem

// UnauthorizedProblemResponse A Problem Details object (RFC 7807)
type UnauthorizedProblemResponse = Problem

// UnexpectedProblemResponse A Problem Details object (RFC 7807)
type UnexpectedProblemResponse = Problem

// ListEventsParams defines parameters for ListEvents.
type ListEventsParams struct {
	// From Start date-time in RFC 3339 format.
	// Inclusive.
	From *QueryFrom `form:"from,omitempty" json:"from,omitempty"`

	// To End date-time in RFC 3339 format.
	// Inclusive.
	To *QueryTo `form:"to,omitempty" json:"to,omitempty"`

	// Limit Number of events to return.
	Limit *int `form:"limit,omitempty" json:"limit,omitempty"`
}

// IngestEventsApplicationCloudeventsBatchPlusJSONBody defines parameters for IngestEvents.
type IngestEventsApplicationCloudeventsBatchPlusJSONBody = []Event

// QueryMeterParams defines parameters for QueryMeter.
type QueryMeterParams struct {
	// From Start date-time in RFC 3339 format.
	// Inclusive.
	From *QueryFrom `form:"from,omitempty" json:"from,omitempty"`

	// To End date-time in RFC 3339 format.
	// Inclusive.
	To *QueryTo `form:"to,omitempty" json:"to,omitempty"`

	// WindowSize If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.
	WindowSize *QueryWindowSize `form:"windowSize,omitempty" json:"windowSize,omitempty"`

	// WindowTimeZone The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
	// If not specified, the UTC timezone will be used.
	WindowTimeZone *QueryWindowTimeZone `form:"windowTimeZone,omitempty" json:"windowTimeZone,omitempty"`
	Subject        *QuerySubject        `form:"subject,omitempty" json:"subject,omitempty"`

	// GroupBy If not specified a single aggregate will be returned for each subject and time window.
	// `subject` is a reserved group by value.
	GroupBy *QueryGroupBy `form:"groupBy,omitempty" json:"groupBy,omitempty"`
}

// QueryPortalMeterParams defines parameters for QueryPortalMeter.
type QueryPortalMeterParams struct {
	// From Start date-time in RFC 3339 format.
	// Inclusive.
	From *QueryFrom `form:"from,omitempty" json:"from,omitempty"`

	// To End date-time in RFC 3339 format.
	// Inclusive.
	To *QueryTo `form:"to,omitempty" json:"to,omitempty"`

	// WindowSize If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group.
	WindowSize *QueryWindowSize `form:"windowSize,omitempty" json:"windowSize,omitempty"`

	// WindowTimeZone The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
	// If not specified, the UTC timezone will be used.
	WindowTimeZone *QueryWindowTimeZone `form:"windowTimeZone,omitempty" json:"windowTimeZone,omitempty"`

	// GroupBy If not specified a single aggregate will be returned for each subject and time window.
	// `subject` is a reserved group by value.
	GroupBy *QueryGroupBy `form:"groupBy,omitempty" json:"groupBy,omitempty"`
}

// ListPortalTokensParams defines parameters for ListPortalTokens.
type ListPortalTokensParams struct {
	// Limit Number of portal tokens to return. Default is 25.
	Limit *int `form:"limit,omitempty" json:"limit,omitempty"`
}

// InvalidatePortalTokensJSONBody defines parameters for InvalidatePortalTokens.
type InvalidatePortalTokensJSONBody struct {
	Subject *string `json:"subject,omitempty"`
}

// IngestEventsApplicationCloudeventsPlusJSONRequestBody defines body for IngestEvents for application/cloudevents+json ContentType.
type IngestEventsApplicationCloudeventsPlusJSONRequestBody = Event

// IngestEventsApplicationCloudeventsBatchPlusJSONRequestBody defines body for IngestEvents for application/cloudevents-batch+json ContentType.
type IngestEventsApplicationCloudeventsBatchPlusJSONRequestBody = IngestEventsApplicationCloudeventsBatchPlusJSONBody

// CreateMeterJSONRequestBody defines body for CreateMeter for application/json ContentType.
type CreateMeterJSONRequestBody = Meter

// CreatePortalTokenJSONRequestBody defines body for CreatePortalToken for application/json ContentType.
type CreatePortalTokenJSONRequestBody = PortalToken

// InvalidatePortalTokensJSONRequestBody defines body for InvalidatePortalTokens for application/json ContentType.
type InvalidatePortalTokensJSONRequestBody InvalidatePortalTokensJSONBody

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

	// (GET /api/v1/portal/meters/{meterSlug}/query)
	QueryPortalMeter(w http.ResponseWriter, r *http.Request, meterSlug string, params QueryPortalMeterParams)

	// (GET /api/v1/portal/tokens)
	ListPortalTokens(w http.ResponseWriter, r *http.Request, params ListPortalTokensParams)

	// (POST /api/v1/portal/tokens)
	CreatePortalToken(w http.ResponseWriter, r *http.Request)

	// (POST /api/v1/portal/tokens/invalidate)
	InvalidatePortalTokens(w http.ResponseWriter, r *http.Request)
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

// (GET /api/v1/portal/meters/{meterSlug}/query)
func (_ Unimplemented) QueryPortalMeter(w http.ResponseWriter, r *http.Request, meterSlug string, params QueryPortalMeterParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (GET /api/v1/portal/tokens)
func (_ Unimplemented) ListPortalTokens(w http.ResponseWriter, r *http.Request, params ListPortalTokensParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (POST /api/v1/portal/tokens)
func (_ Unimplemented) CreatePortalToken(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// (POST /api/v1/portal/tokens/invalidate)
func (_ Unimplemented) InvalidatePortalTokens(w http.ResponseWriter, r *http.Request) {
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

	// ------------- Optional query parameter "windowTimeZone" -------------

	err = runtime.BindQueryParameter("form", true, false, "windowTimeZone", r.URL.Query(), &params.WindowTimeZone)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "windowTimeZone", Err: err})
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

// QueryPortalMeter operation middleware
func (siw *ServerInterfaceWrapper) QueryPortalMeter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "meterSlug" -------------
	var meterSlug string

	err = runtime.BindStyledParameterWithLocation("simple", false, "meterSlug", runtime.ParamLocationPath, chi.URLParam(r, "meterSlug"), &meterSlug)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "meterSlug", Err: err})
		return
	}

	ctx = context.WithValue(ctx, PortalTokenScopes, []string{})

	// Parameter object where we will unmarshal all parameters from the context
	var params QueryPortalMeterParams

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

	// ------------- Optional query parameter "windowTimeZone" -------------

	err = runtime.BindQueryParameter("form", true, false, "windowTimeZone", r.URL.Query(), &params.WindowTimeZone)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "windowTimeZone", Err: err})
		return
	}

	// ------------- Optional query parameter "groupBy" -------------

	err = runtime.BindQueryParameter("form", true, false, "groupBy", r.URL.Query(), &params.GroupBy)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "groupBy", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.QueryPortalMeter(w, r, meterSlug, params)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// ListPortalTokens operation middleware
func (siw *ServerInterfaceWrapper) ListPortalTokens(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params ListPortalTokensParams

	// ------------- Optional query parameter "limit" -------------

	err = runtime.BindQueryParameter("form", true, false, "limit", r.URL.Query(), &params.Limit)
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "limit", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.ListPortalTokens(w, r, params)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// CreatePortalToken operation middleware
func (siw *ServerInterfaceWrapper) CreatePortalToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.CreatePortalToken(w, r)
	}))

	for i := len(siw.HandlerMiddlewares) - 1; i >= 0; i-- {
		handler = siw.HandlerMiddlewares[i](handler)
	}

	handler.ServeHTTP(w, r.WithContext(ctx))
}

// InvalidatePortalTokens operation middleware
func (siw *ServerInterfaceWrapper) InvalidatePortalTokens(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.InvalidatePortalTokens(w, r)
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
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/portal/meters/{meterSlug}/query", wrapper.QueryPortalMeter)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/portal/tokens", wrapper.ListPortalTokens)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/api/v1/portal/tokens", wrapper.CreatePortalToken)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/api/v1/portal/tokens/invalidate", wrapper.InvalidatePortalTokens)
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
