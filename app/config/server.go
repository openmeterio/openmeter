package config

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/server"
)

// ServerConfig holds HTTP server timeout configuration.
type ServerConfig struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration

	ResponseValidation ResponseValidationConfig

	ClientIPMiddleware ClientIPMiddlewareConfig
}

// ResponseValidationConfig controls optional post-response OpenAPI validation on the v3 API.
type ResponseValidationConfig struct {
	Mode ResponseValidationMode
}

type ResponseValidationMode string

const (
	// ResponseValidationModeOff disables response validation. This is the default.
	ResponseValidationModeOff ResponseValidationMode = "off"
	// ResponseValidationModeUnstable validates only routes marked x-unstable: true in the spec.
	ResponseValidationModeUnstable ResponseValidationMode = "unstable"
	// ResponseValidationModeAll validates every route in the v3 spec.
	ResponseValidationModeAll ResponseValidationMode = "all"
)

func (m ResponseValidationMode) Enabled() bool {
	return m != "" && m != ResponseValidationModeOff
}

func (m ResponseValidationMode) Validate() error {
	switch m {
	case "", ResponseValidationModeOff, ResponseValidationModeUnstable, ResponseValidationModeAll:
		return nil
	default:
		return errors.New("invalid response validation mode (allowed: off, unstable, all)")
	}
}

type ClientIPSource string

const (
	ClientIPSourceRemoteAddr ClientIPSource = "remote-address"
	ClientIPSourceHeader     ClientIPSource = "header"
	ClientIPSourceXFF        ClientIPSource = "x-forwarded-for"
)

var _ models.Validator = (*ClientIPMiddlewareConfig)(nil)

// ClientIPMiddlewareConfig configures the middleware that extracts the client IP address from the HTTP request.
// See: https://adam-p.ca/blog/2022/03/x-forwarded-for/
type ClientIPMiddlewareConfig struct {
	Source ClientIPSource

	// Header defines the header name in the HTTP request containing the real client IP address.
	// Set this only if the ClientIPSource is used as Source.
	// E.g. "X-Real-IP" (Nginx), "CF-Connecting-IP" (Cloudflare), or "True-Client-IP" (Cloudflare and Akamai).
	Header string

	// TrustedIPPrefixes lists IP prefixes for trusted proxies.
	// Set this only if the ClientIPSourceXFF is used as Source.
	TrustedIPPrefixes []string

	// TrustedProxies defines the number of trusted proxies.
	// Set this only if the ClientIPSourceXFF is used as Source.
	TrustedProxies int
}

func (c ClientIPMiddlewareConfig) Validate() error {
	switch c.Source {
	case ClientIPSourceRemoteAddr:
		return nil
	case ClientIPSourceHeader:
		if c.Header == "" {
			return fmt.Errorf("missing client IP header")
		}

		return nil
	case ClientIPSourceXFF:
		if len(c.TrustedIPPrefixes) > 0 {
			var invalidPrefixes []string

			for _, prefix := range c.TrustedIPPrefixes {
				if _, _, err := net.ParseCIDR(prefix); err != nil {
					invalidPrefixes = append(invalidPrefixes, prefix)
				}
			}

			if len(invalidPrefixes) > 0 {
				return fmt.Errorf("invalid trusted IP prefixes: %+v", invalidPrefixes)
			}

			return nil
		}

		if c.TrustedProxies == 0 {
			return fmt.Errorf("either trusted IP prefixes or number of trusted proxies must be set if real client IP source is set to %s", ClientIPSourceXFF)
		}

		return nil
	default:
		return fmt.Errorf("invalid client IP source: %s", c.Source)
	}
}

func NewClientIPMiddleware(config ClientIPMiddlewareConfig) (server.MiddlewareFunc, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid client ip middleware config: %w", err)
	}

	switch config.Source {
	case ClientIPSourceRemoteAddr:
		return middleware.ClientIPFromRemoteAddr, nil
	case ClientIPSourceHeader:
		return middleware.ClientIPFromHeader(config.Header), nil
	case ClientIPSourceXFF:
		if len(config.TrustedIPPrefixes) > 0 {
			return middleware.ClientIPFromXFF(config.TrustedIPPrefixes...), nil
		}

		return middleware.ClientIPFromXFFTrustedProxies(config.TrustedProxies), nil
	default:
		return nil, fmt.Errorf("invalid client ip middleware source: %s", config.Source)
	}
}

func (c ServerConfig) Validate() error {
	var errs []error

	if c.ReadHeaderTimeout < 0 {
		errs = append(errs, errors.New("readHeaderTimeout must be non-negative"))
	}

	if c.ReadTimeout < 0 {
		errs = append(errs, errors.New("readTimeout must be non-negative"))
	}

	if c.WriteTimeout < 0 {
		errs = append(errs, errors.New("writeTimeout must be non-negative"))
	}

	if c.IdleTimeout < 0 {
		errs = append(errs, errors.New("idleTimeout must be non-negative"))
	}

	if err := c.ResponseValidation.Mode.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := c.ClientIPMiddleware.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// ConfigureServer sets defaults for HTTP server timeouts.
func ConfigureServer(v *viper.Viper, prefixes ...string) {
	prefixer := NewViperKeyPrefixer(prefixes...)

	v.SetDefault(prefixer("readHeaderTimeout"), 10*time.Second)
	v.SetDefault(prefixer("readTimeout"), 60*time.Second)
	v.SetDefault(prefixer("writeTimeout"), 90*time.Second)
	v.SetDefault(prefixer("idleTimeout"), 120*time.Second)

	v.SetDefault(prefixer("responseValidation.mode"), string(ResponseValidationModeOff))

	v.SetDefault(prefixer("clientIPMiddleware.source"), ClientIPSourceRemoteAddr)
	v.SetDefault(prefixer("clientIPMiddleware.header"), "")
	v.SetDefault(prefixer("clientIPMiddleware.trustedIPPrefixes"), nil)
	v.SetDefault(prefixer("clientIPMiddleware.trustedProxies"), 0)
}
