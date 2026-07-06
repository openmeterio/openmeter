// Package openmeter is a hand-written, idiomatic Go SDK for the OpenMeter v3 API.
//
// It is deliberately shaped as a reference implementation: the public surface
// depends only on the standard library (net/http, context, typed request and
// response structs, and a typed APIError). Retries are provided by an internal
// dependency that is fully hidden behind the *http.Client seam, so callers can
// replace the transport with WithHTTPClient without observing any third-party
// types. This shape is intended to be reproduced by a TypeSpec emitter.
package openmeter

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Client is the entry point to the OpenMeter API. Construct it with New and
// access resources through its grouped sub-clients, e.g. client.Meters.List.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	token      string
	userAgent  string

	// Meters groups the meter operations (get, list, query).
	Meters *MetersService
}

// New creates a Client targeting baseURL, which must include the API version
// prefix (e.g. "https://openmeter.cloud/api/v3").
//
// By default requests go through an internal retrying *http.Client. Provide
// WithHTTPClient to supply your own client and take full ownership of retry,
// timeout, and transport behavior.
func New(baseURL string, opts ...Option) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("openmeter: baseURL is required")
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("openmeter: invalid baseURL %q: %w", baseURL, err)
	}

	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("openmeter: baseURL %q must be absolute (scheme and host)", baseURL)
	}

	c := &Client{
		baseURL:   u,
		userAgent: defaultUserAgent,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient == nil {
		c.httpClient = defaultHTTPClient()
	}

	c.Meters = &MetersService{client: c}

	return c, nil
}

// resolve joins the client base URL with an API path, preserving any base path
// prefix present on the base URL (e.g. a reverse-proxy mount point).
func (c *Client) resolve(apiPath string) *url.URL {
	ref := &url.URL{Path: strings.TrimPrefix(apiPath, "/")}
	base := *c.baseURL

	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}

	return base.ResolveReference(ref)
}

// resourcePath joins a collection base path with an escaped resource ID, or
// returns ErrEmptyID if id is empty. It centralizes the empty-ID guard and URL
// escaping shared by every operation that targets a single resource by ID.
func resourcePath(base, id string) (string, error) {
	if id == "" {
		return "", ErrEmptyID
	}

	return base + "/" + url.PathEscape(id), nil
}
