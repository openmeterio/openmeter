package openmeter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	contentTypeJSON = "application/json"
	contentTypeCSV  = "text/csv"

	// defaultRequestTimeout bounds a buffered request (all retries included) when
	// the caller's context carries no deadline, so a call can't hang forever by
	// default. It is applied via context, not http.Client.Timeout, so it never
	// interferes with streaming body reads. Callers wanting a different bound pass
	// their own context deadline; streaming requests (QueryCSVStream) are never
	// bounded by this and rely solely on the caller's context.
	defaultRequestTimeout = 30 * time.Second

	// maxBufferedResponse caps how much of a response the buffered read paths
	// (JSON decoding, QueryCSV) hold in memory, guarding against unbounded
	// growth from an unexpectedly large payload. Large CSV exports that may
	// exceed this should use MetersService.QueryCSVStream.
	maxBufferedResponse = 10 << 20 // 10 MiB
	// maxErrorBody caps how much of a non-2xx body is read to build an APIError.
	maxErrorBody = 1 << 20 // 1 MiB
)

// defaultHTTPClient builds the SDK's default transport: an internally retrying
// client with exponential backoff, exposed as a standard *http.Client so the
// retry dependency never appears on the SDK's public surface.
//
// It deliberately sets no http.Client.Timeout: that field also bounds reading
// the response body and would abort a streamed export mid-read. Per-call
// deadlines come from the request context instead (see defaultRequestTimeout).
func defaultHTTPClient() *http.Client {
	rc := retryablehttp.NewClient()

	rc.RetryMax = 3
	rc.RetryWaitMin = 500 * time.Millisecond
	rc.RetryWaitMax = 5 * time.Second
	rc.Logger = nil // silence the default stdout logger
	rc.CheckRetry = retryIdempotentOnly
	// Surface the last response when retries are exhausted instead of
	// retryablehttp's default "giving up after N attempts" error, so a retried
	// 5xx still reaches the caller as a typed *APIError. Genuine transport
	// errors (no response) still pass through as an error.
	rc.ErrorHandler = retryablehttp.PassthroughErrorHandler

	return rc.StandardClient()
}

// methodContextKey carries the request method on the request context so the
// retry policy can see it even for a transport error, where no response (and
// thus no resp.Request) is available to read the method from.
type methodContextKey struct{}

func withRequestMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, methodContextKey{}, method)
}

func methodFromContext(ctx context.Context) string {
	method, _ := ctx.Value(methodContextKey{}).(string)
	return method
}

// retryIdempotentOnly restricts automatic retries to idempotent methods (GET,
// HEAD). A non-idempotent request is never retried, since the server may have
// already applied a side effect and a retry could duplicate it. This holds both
// when a response arrived (e.g. a 5xx on POST) and on a transport error
// (resp == nil): a connection can drop after the server processed the request,
// so retrying a POST/PUT/DELETE could still double-apply it. The method comes
// from the response when present, otherwise from the request context (set by
// newRequest), so it is known even when resp is nil. Idempotent (and unknown)
// methods delegate to the default policy.
func retryIdempotentOnly(ctx context.Context, resp *http.Response, err error) (bool, error) {
	method := methodFromContext(ctx)
	if resp != nil && resp.Request != nil {
		method = resp.Request.Method
	}

	switch method {
	case http.MethodGet, http.MethodHead, "":
		// idempotent, or unknown (non-SDK caller) — delegate to the default policy
	default:
		return false, nil
	}

	return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
}

// newRequest builds an *http.Request against the client base URL. body, when
// non-nil, is JSON-encoded and Content-Type is set accordingly. accept sets the
// Accept header (JSON or CSV) to drive server-side content negotiation.
func (c *Client) newRequest(ctx context.Context, method, apiPath string, query url.Values, body any, accept string) (*http.Request, error) {
	u := c.resolve(apiPath)
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("openmeter: encoding request body: %w", err)
		}

		bodyReader = bytes.NewReader(buf)
	}

	// Carry the method on the context so retryIdempotentOnly can honor it even on
	// a transport error, where no response is available to read the method from.
	ctx = withRequestMethod(ctx, method)

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
