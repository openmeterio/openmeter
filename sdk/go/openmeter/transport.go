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
)

// defaultHTTPClient builds the SDK's default transport: an internally retrying
// client with exponential backoff, exposed as a standard *http.Client so the
// retry dependency never appears on the SDK's public surface.
func defaultHTTPClient() *http.Client {
	rc := retryablehttp.NewClient()
	rc.RetryMax = 3
	rc.RetryWaitMin = 500 * time.Millisecond
	rc.RetryWaitMax = 5 * time.Second
	rc.Logger = nil // silence the default stdout logger
	rc.CheckRetry = retryIdempotentOnly
	return rc.StandardClient()
}

// retryIdempotentOnly restricts automatic retries on server responses to
// idempotent methods (GET, HEAD). A non-idempotent request that got a response
// (e.g. a 5xx on POST) is not retried, since the server may have already
// applied a side effect and a retry could duplicate it. When there is no
// response (resp == nil, a transport error before the server replied) the
// method is unknown and the default policy applies — the request most likely
// never reached the server.
func retryIdempotentOnly(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if resp != nil && resp.Request != nil {
		switch resp.Request.Method {
		case http.MethodGet, http.MethodHead:
			// idempotent: fall through to the default policy
		default:
			return false, nil
		}
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

// doRaw executes req, returns the raw 2xx body, and converts any non-2xx
// response into an *APIError.
func (c *Client) doRaw(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openmeter: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openmeter: reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newAPIError(resp.StatusCode, body)
	}

	return body, nil
}
