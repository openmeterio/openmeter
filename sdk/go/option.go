package openmeter

import "net/http"

const defaultUserAgent = "openmeter-go-sdk"

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

// WithHTTPClient replaces the default (internally retrying) *http.Client.
// The provided client owns all transport behavior: retries, timeouts, proxies,
// TLS, and tracing. Pass nil to keep the default.
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
