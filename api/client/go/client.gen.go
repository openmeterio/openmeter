// Package openmeter provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version (devel) DO NOT EDIT.
package openmeter

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/deepmap/oapi-codegen/pkg/runtime"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Event CloudEvents Specification JSON Schema
type Event = event.Event

// Meter defines model for Meter.
type Meter = models.Meter

// MeterAggregation The aggregation type to use for the meter.
type MeterAggregation = models.MeterAggregation

// MeterValue defines model for MeterValue.
type MeterValue = models.MeterValue

// Problem A Problem Details object (RFC 7807)
type Problem = models.Problem

// WindowSize defines model for WindowSize.
type WindowSize = models.WindowSize

// MeterSlug defines model for meterSlug.
type MeterSlug = string

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
	WindowSize *WindowSize `form:"window_size,omitempty" json:"window_size,omitempty"`
}

// IngestEventsJSONRequestBody defines body for IngestEvents for application/cloudevents+json ContentType.
type IngestEventsJSONRequestBody = Event

// CreateMeterJSONRequestBody defines body for CreateMeter for application/json ContentType.
type CreateMeterJSONRequestBody = Meter

// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

// The interface specification for the client above.
type ClientInterface interface {
	// IngestEvents request with any body
	IngestEventsWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	IngestEvents(ctx context.Context, body IngestEventsJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)

	// ListMeters request
	ListMeters(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// CreateMeter request with any body
	CreateMeterWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	CreateMeter(ctx context.Context, body CreateMeterJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)

	// DeleteMeter request
	DeleteMeter(ctx context.Context, meterSlug MeterSlug, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetMeter request
	GetMeter(ctx context.Context, meterSlug MeterSlug, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetMeterValues request
	GetMeterValues(ctx context.Context, meterSlug MeterSlug, params *GetMeterValuesParams, reqEditors ...RequestEditorFn) (*http.Response, error)
}

func (c *Client) IngestEventsWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewIngestEventsRequestWithBody(c.Server, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) IngestEvents(ctx context.Context, body IngestEventsJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewIngestEventsRequest(c.Server, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) ListMeters(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewListMetersRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) CreateMeterWithBody(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewCreateMeterRequestWithBody(c.Server, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) CreateMeter(ctx context.Context, body CreateMeterJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewCreateMeterRequest(c.Server, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) DeleteMeter(ctx context.Context, meterSlug MeterSlug, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewDeleteMeterRequest(c.Server, meterSlug)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetMeter(ctx context.Context, meterSlug MeterSlug, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetMeterRequest(c.Server, meterSlug)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetMeterValues(ctx context.Context, meterSlug MeterSlug, params *GetMeterValuesParams, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetMeterValuesRequest(c.Server, meterSlug, params)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewIngestEventsRequest calls the generic IngestEvents builder with application/cloudevents+json body
func NewIngestEventsRequest(server string, body IngestEventsJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewIngestEventsRequestWithBody(server, "application/cloudevents+json", bodyReader)
}

// NewIngestEventsRequestWithBody generates requests for IngestEvents with any type of body
func NewIngestEventsRequestWithBody(server string, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/api/v1betav1/events")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	return req, nil
}

// NewListMetersRequest generates requests for ListMeters
func NewListMetersRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/api/v1betav1/meters")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewCreateMeterRequest calls the generic CreateMeter builder with application/json body
func NewCreateMeterRequest(server string, body CreateMeterJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewCreateMeterRequestWithBody(server, "application/json", bodyReader)
}

// NewCreateMeterRequestWithBody generates requests for CreateMeter with any type of body
func NewCreateMeterRequestWithBody(server string, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/api/v1betav1/meters")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	return req, nil
}

// NewDeleteMeterRequest generates requests for DeleteMeter
func NewDeleteMeterRequest(server string, meterSlug MeterSlug) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "meterSlug", runtime.ParamLocationPath, meterSlug)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/api/v1betav1/meters/%s", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("DELETE", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetMeterRequest generates requests for GetMeter
func NewGetMeterRequest(server string, meterSlug MeterSlug) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "meterSlug", runtime.ParamLocationPath, meterSlug)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/api/v1betav1/meters/%s", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetMeterValuesRequest generates requests for GetMeterValues
func NewGetMeterValuesRequest(server string, meterSlug MeterSlug, params *GetMeterValuesParams) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "meterSlug", runtime.ParamLocationPath, meterSlug)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/api/v1betav1/meters/%s/values", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	queryValues := queryURL.Query()

	if params.Subject != nil {

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "subject", runtime.ParamLocationQuery, *params.Subject); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

	}

	if params.From != nil {

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "from", runtime.ParamLocationQuery, *params.From); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

	}

	if params.To != nil {

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "to", runtime.ParamLocationQuery, *params.To); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

	}

	if params.WindowSize != nil {

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "window_size", runtime.ParamLocationQuery, *params.WindowSize); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

	}

	queryURL.RawQuery = queryValues.Encode()

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ClientWithResponses builds on ClientInterface to offer response payloads
type ClientWithResponses struct {
	ClientInterface
}

// NewClientWithResponses creates a new ClientWithResponses, which wraps
// Client with return type handling
func NewClientWithResponses(server string, opts ...ClientOption) (*ClientWithResponses, error) {
	client, err := NewClient(server, opts...)
	if err != nil {
		return nil, err
	}
	return &ClientWithResponses{client}, nil
}

// WithBaseURL overrides the baseURL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		newBaseURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		c.Server = newBaseURL.String()
		return nil
	}
}

// ClientWithResponsesInterface is the interface specification for the client with responses above.
type ClientWithResponsesInterface interface {
	// IngestEvents request with any body
	IngestEventsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*IngestEventsResponse, error)

	IngestEventsWithResponse(ctx context.Context, body IngestEventsJSONRequestBody, reqEditors ...RequestEditorFn) (*IngestEventsResponse, error)

	// ListMeters request
	ListMetersWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*ListMetersResponse, error)

	// CreateMeter request with any body
	CreateMeterWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*CreateMeterResponse, error)

	CreateMeterWithResponse(ctx context.Context, body CreateMeterJSONRequestBody, reqEditors ...RequestEditorFn) (*CreateMeterResponse, error)

	// DeleteMeter request
	DeleteMeterWithResponse(ctx context.Context, meterSlug MeterSlug, reqEditors ...RequestEditorFn) (*DeleteMeterResponse, error)

	// GetMeter request
	GetMeterWithResponse(ctx context.Context, meterSlug MeterSlug, reqEditors ...RequestEditorFn) (*GetMeterResponse, error)

	// GetMeterValues request
	GetMeterValuesWithResponse(ctx context.Context, meterSlug MeterSlug, params *GetMeterValuesParams, reqEditors ...RequestEditorFn) (*GetMeterValuesResponse, error)
}

type IngestEventsResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON400      *Problem
	JSONDefault  *Problem
}

// Status returns HTTPResponse.Status
func (r IngestEventsResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r IngestEventsResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type ListMetersResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *[]Meter
	JSONDefault  *Problem
}

// Status returns HTTPResponse.Status
func (r ListMetersResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r ListMetersResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type CreateMeterResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON201      *Meter
	JSON400      *Problem
	JSON405      *Problem
	JSONDefault  *Problem
}

// Status returns HTTPResponse.Status
func (r CreateMeterResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r CreateMeterResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type DeleteMeterResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON404      *Problem
	JSON405      *Problem
	JSONDefault  *Problem
}

// Status returns HTTPResponse.Status
func (r DeleteMeterResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r DeleteMeterResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetMeterResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *Meter
	JSON404      *Problem
	JSONDefault  *Problem
}

// Status returns HTTPResponse.Status
func (r GetMeterResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetMeterResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetMeterValuesResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *struct {
		Data       []MeterValue `json:"data"`
		WindowSize *WindowSize  `json:"windowSize,omitempty"`
	}
	JSON400     *Problem
	JSONDefault *Problem
}

// Status returns HTTPResponse.Status
func (r GetMeterValuesResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetMeterValuesResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// IngestEventsWithBodyWithResponse request with arbitrary body returning *IngestEventsResponse
func (c *ClientWithResponses) IngestEventsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*IngestEventsResponse, error) {
	rsp, err := c.IngestEventsWithBody(ctx, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseIngestEventsResponse(rsp)
}

func (c *ClientWithResponses) IngestEventsWithResponse(ctx context.Context, body IngestEventsJSONRequestBody, reqEditors ...RequestEditorFn) (*IngestEventsResponse, error) {
	rsp, err := c.IngestEvents(ctx, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseIngestEventsResponse(rsp)
}

// ListMetersWithResponse request returning *ListMetersResponse
func (c *ClientWithResponses) ListMetersWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*ListMetersResponse, error) {
	rsp, err := c.ListMeters(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseListMetersResponse(rsp)
}

// CreateMeterWithBodyWithResponse request with arbitrary body returning *CreateMeterResponse
func (c *ClientWithResponses) CreateMeterWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*CreateMeterResponse, error) {
	rsp, err := c.CreateMeterWithBody(ctx, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseCreateMeterResponse(rsp)
}

func (c *ClientWithResponses) CreateMeterWithResponse(ctx context.Context, body CreateMeterJSONRequestBody, reqEditors ...RequestEditorFn) (*CreateMeterResponse, error) {
	rsp, err := c.CreateMeter(ctx, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseCreateMeterResponse(rsp)
}

// DeleteMeterWithResponse request returning *DeleteMeterResponse
func (c *ClientWithResponses) DeleteMeterWithResponse(ctx context.Context, meterSlug MeterSlug, reqEditors ...RequestEditorFn) (*DeleteMeterResponse, error) {
	rsp, err := c.DeleteMeter(ctx, meterSlug, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseDeleteMeterResponse(rsp)
}

// GetMeterWithResponse request returning *GetMeterResponse
func (c *ClientWithResponses) GetMeterWithResponse(ctx context.Context, meterSlug MeterSlug, reqEditors ...RequestEditorFn) (*GetMeterResponse, error) {
	rsp, err := c.GetMeter(ctx, meterSlug, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetMeterResponse(rsp)
}

// GetMeterValuesWithResponse request returning *GetMeterValuesResponse
func (c *ClientWithResponses) GetMeterValuesWithResponse(ctx context.Context, meterSlug MeterSlug, params *GetMeterValuesParams, reqEditors ...RequestEditorFn) (*GetMeterValuesResponse, error) {
	rsp, err := c.GetMeterValues(ctx, meterSlug, params, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetMeterValuesResponse(rsp)
}

// ParseIngestEventsResponse parses an HTTP response from a IngestEventsWithResponse call
func ParseIngestEventsResponse(rsp *http.Response) (*IngestEventsResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &IngestEventsResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}

// ParseListMetersResponse parses an HTTP response from a ListMetersWithResponse call
func ParseListMetersResponse(rsp *http.Response) (*ListMetersResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &ListMetersResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest []Meter
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}

// ParseCreateMeterResponse parses an HTTP response from a CreateMeterWithResponse call
func ParseCreateMeterResponse(rsp *http.Response) (*CreateMeterResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &CreateMeterResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 201:
		var dest Meter
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON201 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 405:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON405 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}

// ParseDeleteMeterResponse parses an HTTP response from a DeleteMeterWithResponse call
func ParseDeleteMeterResponse(rsp *http.Response) (*DeleteMeterResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &DeleteMeterResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 404:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON404 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 405:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON405 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}

// ParseGetMeterResponse parses an HTTP response from a GetMeterWithResponse call
func ParseGetMeterResponse(rsp *http.Response) (*GetMeterResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetMeterResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest Meter
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 404:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON404 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}

// ParseGetMeterValuesResponse parses an HTTP response from a GetMeterValuesWithResponse call
func ParseGetMeterValuesResponse(rsp *http.Response) (*GetMeterValuesResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetMeterValuesResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest struct {
			Data       []MeterValue `json:"data"`
			WindowSize *WindowSize  `json:"windowSize,omitempty"`
		}
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 400:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON400 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest Problem
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/9RaaXPbxhn+Kztbf0gakAQPSTa/dCRZTtiaksaikqaWqlkCL8FNgF14dyGJ0fC/d/bA",
	"RYAm5ch1O+PxCHs+730sn3DAk5QzYEri8RNOiSAJKBDmy/x1FWeR/ghBBoKminKGx/gYZYx+ygDREJii",
	"CwoCLbhAagnIbOtiD1O9MiVqiT3MSAJ4XDnSwwI+ZVRAiMdKZOBhGSwhIRaGUiD07n9/JJ0//M6b2x++",
	"+9v4rvj4/q+vsIfVKtVnSiUoi/B6vdZnypQzCQb/CQk/wKcMpLoUfB5D8sHN6smAMwVM6T9JmsY0IJq0",
	"XmpX/vCb1HQ+VUC9ErDAY/yXXsmynp2VPXe+xVDn1AkJkUOB1x6eglry8Jyr4zjmDxB+Q2QWCmJcIWLB",
	"eGgBRGUCzKDM0pQLBaHGfc7VO56xb4n3nCtkMGg81wweUwjUN+VgCQKBEFxgvcTt08ee3ecowpDqPSS+",
	"FDwFoajW0AWJJWyeeRrzLDQbJbpKIaALBx/9/eriHF1ZzB5OKwc94ZAosv0ia1/1e2ZLQKCvQSlZxZyE",
	"2mLhkSRprI3q6YYhdIPDTJjL7xJ5g8foBvcHN9izc9qy7WBvCXHMb/ANW9+w0jD5/DcIjNZreE4odm7T",
	"n5zaSaRnEV8YP6I3oXsSZ9BF00wqRMIlCECKow/vTtHAHx1qn5MQZaCzLMHjjzWBG0HfVslqzHo4oew9",
	"sEgt8bjvYZbFMZnrtZZpGz7GklIqTp2KSe4MpSHALkNqSZQlxhIgkeIascWOxzgT9Pk4aLjzfiPeulgP",
	"gr6/ICF0+sEb6IzCw6DzenB00AkOBsHw8GjYD4dBA0vjbskzEcDO+43EHxWiDD0sabBEhDmVW5I0BQYb",
	"OidB3NMAZM/90fE3uNQRsAABLIA9MKYQ3IOQ1Np6U/ndZK5tVbuTNbuz2AuGokyCrAPvd/09AGXWHBpg",
	"3pqvea40dlkOy15JWY2htblU8DALQKDvilgcovkKWSF9X0caZFLxBMQdDXcjVjRpEfKMJiAVSVIN42EJ",
	"FhoPgkwY0ZTCbbPa4XD4pg5p4A+GHb/f8fszvz82/7q+3/9XVfYhUdAxcJ5tJ+3+ps7z3OtYhgqIiXbp",
	"ilvKBI0oI4qyqEJlnQaS0jvhgvwOtq6ric9HbOTg7KmutG5nqTi3m47Vw4+diHfcoLV2G3IqMx2a6Cju",
	"0qolHuOIqmU27wY86QVa6c1G2ZPh752I9+4HPTNgkE51uma21qINiSIBEVHOtj4XPc0Jx5X1m0G0mVlW",
	"vnNNL1LKkuXTFTJno7eV7XvogyFu1qoUZUg0CqE4ygn9rLgbV0SCZ+nJqnmBjuCXRC0RPKYCpBazDgYI",
	"HpUggTKkms3afk3kk2gheFKxdh1H6mA+2mj8qpus7mIyh1jH4VsTh6mCxAhsi1VgIgRZGef05Vl+iSRZ",
	"dcyozk6+KIP3sCHapS77MXCTf+aE3Vx71a2kNm1IHigL+cMV/QN2qfgv5cpN65a20jHFj1czm6oibpJd",
	"u3yH1Sc8hFh2p47v+5k9T4EZSVFe/t1Lf4969rjS9o/rlt60lwpNhdVkElrUxOVnV9dT7OHTi+vzGfbw",
	"dHKu/z/+Zz5293ZyNZucn+rJ98ezs6vZ3cmvdxfv3l2dzSq8cHLazovjGq9fmi0/a3k1/WLF8NsT8e2W",
	"WObKW/OEWTM7MAhdjrzVnCrXsiyZgyjV+4yZLLI1zm6zCEUs//bZtN5feS1LX1RUeeX2rLroGLlt6C0o",
	"QmOJLHT0nU5fjl77R9+3FUoasS2GMsHGrtI0dIznEEd0MXZl1ozz90REUJRRiqrYbb20S9CMc1RfJBVR",
	"mSnBRv2hGwsNPrvTdRhQAlKSCFCiMy9dxM8BxfokbY2Eob6PpifFqZRJRVhQAZ5lNBz3B0MYHRwedeD1",
	"m3mnPwiHHTI6OOyMBoeH/VH/aOT7fnFGTBOq9AF9f/T64OjQz4vAjQLVgG0LMcssIawjgIQ6bmvPHhNm",
	"HUqeh9tUjMpqlukswHG67tyfxY02Vc8Z04b3+sMEFXWILe7oRt2Xw94T7rMY3ywcm5WGUZZ2B/LTbHaJ",
	"7AIU8BBQBAyEyXfnq0q+i3QNVoT7vVmvtRPDYxBnkt7DlDzSRLt9a2gFbsrUcKAT5Xz+0PdN2my/RvrL",
	"UUWZgsi6LGMobRIhSC65UN6mKsksSYhYbeA1YaougIbd4b0LCR3ATV1GKJOIGO1o04ntl+/pL3bKfSP1",
	"cMWDZVqhE15uiPulFbkHfVG3/Ests8rzgunk/Hp2hj3808X1B+zht8e/7hnvK+e9IM61cQMLrg+IaQCu",
	"tej62McpCZaABqbwz0SMx3ipVDru9R4eHrrEzHa5iHpuq+y9n5yenV+ddQZdv7tUSVzRaHyRArPFzPHl",
	"RKeDed8C97t+1+/MQZFuX+/QYElK8RgPu353aDPtpbH1Hklp776v1973bQ1ne/pctiQTExZpD+mWmYNt",
	"OjwJi9mzfNLVOic8XH2mvVopJp/ZYj1zBed681Fgs6M/8P0mKRf/0JwZ2am2a4ojettfBExZuiBZrHaf",
	"sr33bBIeEkmt0I57t3qsLpvyhSWCFtG8pyZ0mTWbgtFz03yqjTVbZNOUR1Eb7qzem/Vi61sCCInE12On",
	"o/p27W3R6FMBRLmUuME3O5lXSfvp8/N02LFqHx3uf41L25gRvoxhjPyD3Sfsetj6ihqxxcB6T8Vr49qq",
	"SwyqtReox10tNV8hV7TXFcgumhbtjfKd9GM7NeWSXvnoub5tKMOoCeicI/ceYrk/2s2xbc9z/wfS89q9",
	"4I+gdojkR1BfQR7+1zfOPGD9Obl+Q2vq2d7k1ghWys4t3Ca6n/PpLxag92R/ZvApA7Eqf2eQN1Gqvypo",
	"5MubsE1rAxXtDERZ8WSRPzTeMPOmMQdEYhoxCNEDVfZxyLZHkKR/QPeGTZirgbq2E9uCcSF4UgO4X0dl",
	"E/UZC/97mBV/AcSThf19gS2TIfR0BUdZFAPKTL1etN3RA41jjVyAyoSGnvcVdXElQBXFXXEYSnVWbxcC",
	"CZZFy4yw0DbXt1Nn2XEnbSmxn5XXur9/1p20P+rvn6rZJlpLf/9FutkGT6Nw/IyD+x/LyEsXVw62WFPK",
	"KVOy+gxITTlEWVTWS05jXJLfapXNc4qk3u12gNa36/8EAAD//6KH4TWJJQAA",
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
