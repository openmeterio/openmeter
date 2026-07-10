// Static Go SDK runtime files emitted into the generated SDK.
//
// Keep these as emitter-owned TypeScript templates rather than files in a
// runtime/ Go package. That avoids making the emitter source tree look like a
// standalone Go module while still keeping the generated runtime reviewable.

export const RUNTIME_TEMPLATES: Record<string, string> = {
  'errors.go': `package openmeter

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrEmptyID is returned by operations that target a single resource when the
// resource ID is empty. It is caught before any request is made so an omitted
// ID surfaces as a clear client-side error rather than an ambiguous server
// response. Match it with errors.Is.
var ErrEmptyID = errors.New("openmeter: resource ID must not be empty")

// APIError is returned for any non-2xx API response. It mirrors the API's
// RFC 7807-style problem body. When the body cannot be parsed as such, Title is
// left empty and RawBody carries the undecoded payload.
type APIError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int \`json:"-"\`

	// Status is the status code echoed in the problem body (usually equal to
	// StatusCode).
	Status int \`json:"status"\`
	// Title is a short, stable, human-readable summary of the problem.
	Title string \`json:"title"\`
	// Type is an optional machine-readable error type.
	Type string \`json:"type,omitempty"\`
	// Detail is a human-readable explanation specific to this occurrence.
	Detail string \`json:"detail"\`
	// Instance carries the correlation ID, formatted as kong:trace:<id>.
	Instance string \`json:"instance"\`

	// RawBody is the undecoded response body, always populated.
	RawBody []byte \`json:"-"\`
}

func newAPIError(statusCode int, body []byte) *APIError {
	e := &APIError{StatusCode: statusCode, RawBody: body}
	// Best-effort decode; a non-conforming body still yields a useful error via
	// StatusCode and RawBody.
	_ = json.Unmarshal(body, e)
	return e
}

// AsAPIError returns the APIError inside err, when err came from an API response.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// Decode decodes the original error response body into out.
func (e *APIError) Decode(out any) error {
	return json.Unmarshal(e.RawBody, out)
}

// DecodeAPIError decodes an API error body into T. The returned boolean is false
// when err is not an APIError.
func DecodeAPIError[T any](err error) (T, bool, error) {
	var zero T
	apiErr, ok := AsAPIError(err)
	if !ok {
		return zero, false, nil
	}

	var out T
	if err := apiErr.Decode(&out); err != nil {
		return zero, true, err
	}
	return out, true, nil
}

func (e *APIError) Error() string {
	switch {
	case e.Title != "" && e.Detail != "":
		return fmt.Sprintf("openmeter: %d %s: %s", e.StatusCode, e.Title, e.Detail)
	case e.Title != "":
		return fmt.Sprintf("openmeter: %d %s", e.StatusCode, e.Title)
	default:
		// No RFC 7807 fields parsed (e.g. a proxy returned an HTML error page).
		// Inline the raw body for diagnostics but bound it so a large payload
		// can't blow up log lines; RawBody still holds the full response.
		const maxInline = 512
		body := e.RawBody
		suffix := ""
		if len(body) > maxInline {
			body = body[:maxInline]
			suffix = "… (truncated)"
		}
		return fmt.Sprintf("openmeter: unexpected status %d: %s%s", e.StatusCode, string(body), suffix)
	}
}
`,
  'filters.go': `package openmeter

import "time"

// StringFilter expresses the comparison operators accepted for string fields.
type StringFilter struct {
	Eq        *string
	Neq       *string
	Gt        *string
	Gte       *string
	Lt        *string
	Lte       *string
	Contains  *string
	Oeq       []string
	Ocontains []string
	Exists    *bool
}

// StringExactFilter expresses exact comparisons for strings and ULIDs.
type StringExactFilter struct {
	Eq  *string
	Neq *string
	Oeq []string
}

// DateTimeFilter expresses comparisons against RFC 3339 timestamps.
type DateTimeFilter struct {
	Eq  *time.Time
	Gt  *time.Time
	Gte *time.Time
	Lt  *time.Time
	Lte *time.Time
}

// NumericFilter expresses numeric comparison operators.
type NumericFilter struct {
	Eq  *float64
	Neq *float64
	Gt  *float64
	Gte *float64
	Lt  *float64
	Lte *float64
	Oeq []float64
}

// BooleanFilter expresses equality for a boolean field.
type BooleanFilter struct {
	Eq *bool
}
`,
  // {{MODULE_PATH}} is interpolated from the required module-path option.
  // {{GO_VERSION}} comes from the go-version option, defaulting to 1.23 (the
  // generated code's actual floor, needed by the iter package); a repo can
  // raise it to a higher consumer floor, e.g. when preserved *_test.go files
  // need newer stdlib APIs. The single-entry require stays unparenthesized so
  // 'go mod tidy -diff' is a no-op on the emitted file.
  'go.mod': `module {{MODULE_PATH}}

go {{GO_VERSION}}

require github.com/oapi-codegen/nullable v1.2.0
`,
  'go.sum': `github.com/oapi-codegen/nullable v1.2.0 h1:VflFkDW980KhBPiFF7nWSyjg+r4Obqj8lXipV0UkP5w=
github.com/oapi-codegen/nullable v1.2.0/go.mod h1:KUZ3vUzkmEKY90ksAmit2+5juDIhIZhfDl+0PwOQlFY=
`,
  'nullable.go': `package openmeter

import "github.com/oapi-codegen/nullable"

// Nullable represents a JSON field with three states: unspecified, explicit
// null, or a concrete value. Optional nullable fields use this as a value with
// omitempty; do not wrap it in a pointer.
type Nullable[T any] = nullable.Nullable[T]

// Null constructs an explicit JSON null.
func Null[T any]() Nullable[T] {
	return nullable.NewNullNullable[T]()
}

// NullableValue constructs a non-null value.
func NullableValue[T any](value T) Nullable[T] {
	return nullable.NewNullableWithValue(value)
}
`,
  'one_or_many.go': `package openmeter

import (
	"bytes"
	"encoding/json"
)

// OneOrMany represents a JSON value that accepts either one T or an array of T.
// The zero value holds neither variant and marshals as JSON null.
type OneOrMany[T any] struct {
	one  *T
	many []T
}

// One wraps a single value.
func One[T any](value T) OneOrMany[T] {
	return OneOrMany[T]{one: &value}
}

// Many wraps an array. A non-nil empty slice is encoded as [].
func Many[T any](values []T) OneOrMany[T] {
	return OneOrMany[T]{many: values}
}

// AsOne returns the single value. The boolean is false when the array variant
// is set or the value is empty.
func (value OneOrMany[T]) AsOne() (*T, bool) {
	if value.many != nil || value.one == nil {
		return nil, false
	}
	return value.one, true
}

// AsMany returns the array. The boolean is false when the single-value variant
// is set or the value is empty.
func (value OneOrMany[T]) AsMany() ([]T, bool) {
	if value.many == nil {
		return nil, false
	}
	return value.many, true
}

func (value OneOrMany[T]) MarshalJSON() ([]byte, error) {
	if value.many != nil {
		return json.Marshal(value.many)
	}
	if value.one != nil {
		return json.Marshal(value.one)
	}
	return []byte("null"), nil
}

func (value *OneOrMany[T]) UnmarshalJSON(data []byte) error {
	*value = OneOrMany[T]{}
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	if len(data) > 0 && data[0] == '[' {
		return json.Unmarshal(data, &value.many)
	}
	var one T
	if err := json.Unmarshal(data, &one); err != nil {
		return err
	}
	value.one = &one
	return nil
}
`,
  'option.go': `package openmeter

import "net/http"

// Version is the SDK version reported in the default User-Agent. It stays a
// development placeholder until the release process stamps the released value
// via the sdk-version emitter option.
const Version = "{{SDK_VERSION}}"

const defaultUserAgent = "openmeter-go-sdk/" + Version

// Option configures a Client during New.
type Option func(*Client)

// WithToken sets the bearer token sent in the Authorization header of every
// request. The header is applied during request construction, so it is honored
// regardless of any client injected via WithHTTPClient.
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// WithHTTPClient replaces the default *http.Client. The provided client owns all
// transport behavior: retries, timeouts, proxies, TLS, and tracing. Pass nil to
// keep the default.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithUserAgent overrides the User-Agent header sent with each request.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		if ua != "" {
			c.userAgent = ua
		}
	}
}
`,
  'pagination.go': `package openmeter

import (
	"fmt"
	"iter"
	"net/url"
	"strconv"
)

const defaultListPageSize = 100
const maxPages = 10_000

// PageParams selects a numbered page.
type PageParams struct {
	Size   *int
	Number *int
}

// CursorPageParams selects a cursor page.
type CursorPageParams struct {
	Size   *int
	After  *string
	Before *string
}

type PageMeta struct {
	Number int \`json:"number"\`
	Size   int \`json:"size"\`
	Total  int \`json:"total"\`
}

type PaginatedMeta struct {
	Page PageMeta \`json:"page"\`
}

func addPageParams(q url.Values, page *PageParams) {
	if page == nil {
		return
	}
	if page.Size != nil {
		setDeepObjectString(q, "page", "size", strconv.Itoa(*page.Size))
	}
	if page.Number != nil {
		setDeepObjectString(q, "page", "number", strconv.Itoa(*page.Number))
	}
}

func addCursorPageParams(q url.Values, page *CursorPageParams) {
	if page == nil {
		return
	}
	if page.Size != nil {
		setDeepObjectString(q, "page", "size", strconv.Itoa(*page.Size))
	}
	if page.After != nil {
		setDeepObjectString(q, "page", "after", *page.After)
	}
	if page.Before != nil {
		setDeepObjectString(q, "page", "before", *page.Before)
	}
}

func paginate[T any](start *PageParams, fetch func(page, size int) ([]T, int, error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		page, size := 1, defaultListPageSize
		if start != nil {
			if start.Number != nil {
				page = *start.Number
			}
			if start.Size != nil {
				size = *start.Size
			}
		}
		seen := 0
		for fetched := 0; fetched < maxPages; fetched++ {
			data, total, err := fetch(page, size)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			for _, item := range data {
				if !yield(item, nil) {
					return
				}
			}
			seen += len(data)
			if len(data) == 0 || (total > 0 && seen >= total) {
				return
			}
			page++
		}
		var zero T
		yield(zero, fmt.Errorf("openmeter: pagination did not terminate within %d pages", maxPages))
	}
}

func paginateCursor[T any](start *CursorPageParams, fetch func(after, before *string, size int) ([]T, *string, *string, error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		size := defaultListPageSize
		var after, before *string
		if start != nil {
			after = start.After
			before = start.Before
			if after != nil && before != nil {
				var zero T
				yield(zero, fmt.Errorf("openmeter: cursor pagination cannot use both after and before"))
				return
			}
			if start.Size != nil {
				size = *start.Size
			}
		}
		reverse := before != nil
		for fetched := 0; fetched < maxPages; fetched++ {
			data, next, previous, err := fetch(after, before, size)
			if err != nil {
				var zero T
				yield(zero, err)
				return
			}
			for _, item := range data {
				if !yield(item, nil) {
					return
				}
			}
			if reverse {
				if previous == nil || *previous == "" || len(data) == 0 {
					return
				}
				before = previous
			} else {
				if next == nil || *next == "" || len(data) == 0 {
					return
				}
				after = next
			}
		}
		var zero T
		yield(zero, fmt.Errorf("openmeter: cursor pagination did not terminate within %d pages", maxPages))
	}
}
`,
  'ptr.go': `package openmeter

import "time"

// Pointer helpers for populating optional request fields inline. They mirror the
// convention used by other Go cloud SDKs (e.g. aws.String), keeping call sites
// free of one-off address-of locals.

// Ptr returns a pointer to v. It is the generic form covering any type; the
// typed String/Int/Bool/Time below remain because they let the compiler infer
// the element type at call sites where a bare literal would not (e.g.
// String("x") vs Ptr("x"), which are equivalent, but Int(1) avoids Ptr[int](1)).
func Ptr[T any](v T) *T { return &v }

// String returns a pointer to s.
func String(s string) *string { return &s }

// Int returns a pointer to i.
func Int(i int) *int { return &i }

// Bool returns a pointer to b.
func Bool(b bool) *bool { return &b }

// Time returns a pointer to t.
func Time(t time.Time) *time.Time { return &t }
`,
  'query.go': `package openmeter

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SortOrder is the direction of a sort expression.
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// Sort selects a wire field and optional direction.
type Sort struct {
	By    string
	Order SortOrder
}

func setDeepObjectString(q url.Values, prefix, key, value string) {
	q.Set(prefix+"["+key+"]", value)
}

func addSort(q url.Values, name string, sort *Sort) {
	if sort == nil || sort.By == "" {
		return
	}
	value := sort.By
	if sort.Order != "" {
		value += " " + string(sort.Order)
	}
	q.Set(name, value)
}

func addStringFilter(q url.Values, prefix string, f *StringFilter) {
	if f == nil {
		return
	}
	if f.Eq != nil {
		setDeepObjectString(q, prefix, "eq", *f.Eq)
	}
	if f.Neq != nil {
		setDeepObjectString(q, prefix, "neq", *f.Neq)
	}
	if f.Contains != nil {
		setDeepObjectString(q, prefix, "contains", *f.Contains)
	}
	if f.Gt != nil {
		setDeepObjectString(q, prefix, "gt", *f.Gt)
	}
	if f.Gte != nil {
		setDeepObjectString(q, prefix, "gte", *f.Gte)
	}
	if f.Lt != nil {
		setDeepObjectString(q, prefix, "lt", *f.Lt)
	}
	if f.Lte != nil {
		setDeepObjectString(q, prefix, "lte", *f.Lte)
	}
	if len(f.Oeq) > 0 {
		setDeepObjectString(q, prefix, "oeq", strings.Join(f.Oeq, ","))
	}
	if len(f.Ocontains) > 0 {
		setDeepObjectString(q, prefix, "ocontains", strings.Join(f.Ocontains, ","))
	}
	if f.Exists != nil {
		setDeepObjectString(q, prefix, "exists", strconv.FormatBool(*f.Exists))
	}
}

func addStringExactFilter(q url.Values, prefix string, f *StringExactFilter) {
	if f == nil {
		return
	}
	if f.Eq != nil {
		setDeepObjectString(q, prefix, "eq", *f.Eq)
	}
	if f.Neq != nil {
		setDeepObjectString(q, prefix, "neq", *f.Neq)
	}
	if len(f.Oeq) > 0 {
		setDeepObjectString(q, prefix, "oeq", strings.Join(f.Oeq, ","))
	}
}

func addDateTimeFilter(q url.Values, prefix string, f *DateTimeFilter) {
	if f == nil {
		return
	}
	values := []struct {
		name  string
		value *time.Time
	}{
		{"eq", f.Eq},
		{"gt", f.Gt},
		{"gte", f.Gte},
		{"lt", f.Lt},
		{"lte", f.Lte},
	}
	for _, value := range values {
		if value.value != nil {
			setDeepObjectString(q, prefix, value.name, value.value.Format(time.RFC3339Nano))
		}
	}
}

func addNumericFilter(q url.Values, prefix string, f *NumericFilter) {
	if f == nil {
		return
	}
	format := func(value float64) string {
		return strconv.FormatFloat(value, 'g', -1, 64)
	}
	values := []struct {
		name  string
		value *float64
	}{
		{"eq", f.Eq},
		{"neq", f.Neq},
		{"gt", f.Gt},
		{"gte", f.Gte},
		{"lt", f.Lt},
		{"lte", f.Lte},
	}
	for _, value := range values {
		if value.value != nil {
			setDeepObjectString(q, prefix, value.name, format(*value.value))
		}
	}
	if len(f.Oeq) > 0 {
		values := make([]string, 0, len(f.Oeq))
		for _, value := range f.Oeq {
			values = append(values, format(value))
		}
		setDeepObjectString(q, prefix, "oeq", strings.Join(values, ","))
	}
}

func addBooleanFilter(q url.Values, prefix string, f *BooleanFilter) {
	if f != nil && f.Eq != nil {
		setDeepObjectString(q, prefix, "eq", strconv.FormatBool(*f.Eq))
	}
}
`,
  'request_content_type.go': `package openmeter

import (
	"context"
	"net/http"
	"net/url"
)

func (c *Client) newRequestWithContentType(
	ctx context.Context,
	method string,
	apiPath string,
	query url.Values,
	body any,
	contentType string,
	accept string,
) (*http.Request, error) {
	req, err := c.newRequest(ctx, method, apiPath, query, body, accept)
	if err != nil {
		return nil, err
	}
	if body != nil && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

func optionalBody[T any](body *T) any {
	if body == nil {
		return nil
	}
	return body
}
`,
  'transport.go': `package openmeter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	contentTypeJSON = "application/json"

	// defaultRequestTimeout bounds a buffered request when the caller's context
	// carries no deadline, so a call can't hang forever by default. It is applied
	// via context, not http.Client.Timeout, so it never interferes with streaming
	// body reads. Callers wanting a different bound pass their own context
	// deadline; requests made through Stream method variants are never bounded by
	// this and rely solely on the caller's context.
	defaultRequestTimeout = 30 * time.Second

	// maxBufferedResponse caps how much of a response the buffered read paths
	// (JSON decoding, byte-returning text methods) hold in memory, guarding
	// against unbounded growth from an unexpectedly large payload. Exports that
	// may exceed this should use the operation's Stream method variant.
	maxBufferedResponse = 10 << 20 // 10 MiB
	// maxErrorBody caps how much of a non-2xx body is read to build an APIError.
	maxErrorBody = 1 << 20 // 1 MiB
)

// defaultHTTPClient builds the SDK's default transport.
//
// It deliberately sets no http.Client.Timeout: that field also bounds reading the
// response body and would abort a streamed export mid-read. Per-call deadlines
// come from the request context instead (see defaultRequestTimeout).
func defaultHTTPClient() *http.Client {
	return &http.Client{}
}

// newRequest builds an *http.Request against the client base URL. body, when
// non-nil, is JSON-encoded and Content-Type is set accordingly. accept sets the
// Accept header (JSON or CSV) to drive server-side content negotiation.
func (c *Client) newRequest(ctx context.Context, method, apiPath string, query url.Values, body any, accept string) (*http.Request, error) {
	u := c.resolve(apiPath)
	if len(query) > 0 {
		merged := u.Query()
		for key, values := range query {
			merged.Del(key)
			for _, value := range values {
				merged.Add(key, value)
			}
		}
		u.RawQuery = merged.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("openmeter: encoding request body: %w", err)
		}

		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("openmeter: building request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", contentTypeJSON)
	}

	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return req, nil
}

// doJSON executes req and decodes a 2xx JSON body into out (out may be nil to
// discard the body). Non-2xx responses are converted to *APIError.
func (c *Client) doJSON(req *http.Request, out any) error {
	body, err := c.doRaw(req)
	if err != nil {
		return err
	}

	if out == nil || len(body) == 0 {
		return nil
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("openmeter: decoding response body: %w", err)
	}

	return nil
}

// withDefaultDeadline bounds a buffered request to defaultRequestTimeout when
// the caller's context carries no deadline, so a call can't hang forever by
// default. When the caller already set a deadline, the request is returned
// unchanged. The returned cancel func must always be called; it is a no-op in
// the pass-through case. Streaming requests intentionally skip this so a long
// body read is bounded only by the caller's own context.
func withDefaultDeadline(req *http.Request) (*http.Request, context.CancelFunc) {
	if _, ok := req.Context().Deadline(); ok {
		return req, func() {}
	}

	ctx, cancel := context.WithTimeout(req.Context(), defaultRequestTimeout)
	return req.WithContext(ctx), cancel
}

// doRaw executes req, returns the 2xx body (capped at maxBufferedResponse), and
// converts any non-2xx response into an *APIError. Use doStream for responses
// that may exceed the buffered limit (e.g. large CSV exports).
func (c *Client) doRaw(req *http.Request) ([]byte, error) {
	req, cancel := withDefaultDeadline(req)
	defer cancel()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openmeter: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := readAllCapped(resp.Body, maxErrorBody)
		return nil, newAPIError(resp.StatusCode, body)
	}

	body, err := readAllCapped(resp.Body, maxBufferedResponse)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// doStream executes req and returns the live response for streaming. The caller
// owns resp.Body and must close it. Non-2xx responses are converted to
// *APIError (with the body closed) exactly as the buffered paths do, so a
// successful return always carries a readable body.
func (c *Client) doStream(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openmeter: request failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()

		body, _ := readAllCapped(resp.Body, maxErrorBody)

		return nil, newAPIError(resp.StatusCode, body)
	}

	return resp, nil
}

// readAllCapped reads up to max bytes from r and returns an error if the source
// carries more, bounding how much a buffered response can hold in memory. On
// error it still returns whatever bytes were read (capped at max) so callers can
// preserve partial diagnostic content, e.g. an oversized or truncated error body.
func readAllCapped(r io.Reader, max int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return body, fmt.Errorf("openmeter: reading response body: %w", err)
	}

	if int64(len(body)) > max {
		return body[:max], fmt.Errorf("openmeter: response body exceeds %d-byte limit; use a streaming method for large payloads", max)
	}

	return body, nil
}
`,
  'types.go': `package openmeter

// Numeric represents an arbitrary-precision number. The API encodes it as a
// decimal string (e.g. "12.3456") to avoid float precision loss, so the SDK
// surfaces it as a string. Parse with a decimal library when arithmetic is needed.
type Numeric = string
`,
}
