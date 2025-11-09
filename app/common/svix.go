package common

import (
	"crypto/tls"
	"net/http"
	"net/url"

	svix "github.com/svix/svix-webhooks/go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
)

func NewSvixAPIClient(
	config config.SvixConfig,
	meterProvider metric.MeterProvider,
	tracerProvider trace.TracerProvider,
) (*svix.Svix, error) {
	if !config.IsEnabled() {
		return nil, nil
	}

	opts := &svix.SvixOptions{
		Debug: config.Debug,
	}

	if config.ServerURL != "" {
		serverURL, err := url.Parse(config.ServerURL)
		if err != nil {
			return nil, err
		}

		opts.ServerUrl = serverURL
	}

	// See: https://github.com/svix/svix-webhooks/blob/main/go/internal/svix_http_client.go#L31
	// Disable HTTP/2.0
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ForceAttemptHTTP2 = false
	tr.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	tr.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)

	opts.HTTPClient = &http.Client{
		Transport: otelhttp.NewTransport(
			tr,
			otelhttp.WithMeterProvider(meterProvider),
			otelhttp.WithTracerProvider(tracerProvider),
			otelhttp.WithSpanOptions(trace.WithAttributes(semconv.PeerService("svix"))),
		),
	}

	return svix.New(config.APIKey, opts)
}
