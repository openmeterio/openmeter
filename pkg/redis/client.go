package redis

import (
	"crypto/tls"
	"fmt"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Option func(*Options)

// WithTracingProvider allows to instrument redis.Client with custom tracing provider
func WithTracingProvider(p trace.TracerProvider) Option {
	return func(o *Options) {
		if p != nil {
			o.TracingProvider = p
		}
	}
}

// WithMeterProvider allows to instrument redis.Client with custom metrics provider
func WithMeterProvider(p metric.MeterProvider) Option {
	return func(o *Options) {
		if p != nil {
			o.MeterProvider = p
		}
	}
}

// Options stores all the input parameters to initialize new redis.Client
type Options struct {
	Config

	TracingProvider trace.TracerProvider
	MeterProvider   metric.MeterProvider
}

// NewClient returns a new redis.Client initialized by using configuration parameters provided in Options.
func NewClient(o Options, opts ...Option) (*redis.Client, error) {
	// Apply extra options
	for _, opt := range opts {
		opt(&o)
	}

	// Setup TLS if enabled
	var tlsConfig *tls.Config
	if o.TLS.Enabled {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: o.TLS.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS13,
		}
	}

	// Initialize Redis Client
	var client *redis.Client
	if o.Sentinel.Enabled {
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    o.Sentinel.MasterName,
			SentinelAddrs: []string{o.Address},
			DB:            o.Database,
			Username:      o.Username,
			Password:      o.Password,
			TLSConfig:     tlsConfig,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:      o.Address,
			DB:        o.Database,
			Username:  o.Username,
			Password:  o.Password,
			TLSConfig: tlsConfig,
		})
	}

	// Enable tracing
	var tracingOpts []redisotel.TracingOption
	if o.TracingProvider != nil {
		tracingOpts = append(tracingOpts, redisotel.WithTracerProvider(o.TracingProvider))
	}
	if err := redisotel.InstrumentTracing(client, tracingOpts...); err != nil {
		return nil, fmt.Errorf("failed to instrument redis client with tracing provider: %w", err)
	}

	// Enable metrics
	var metricsOpts []redisotel.MetricsOption
	if o.MeterProvider != nil {
		metricsOpts = append(metricsOpts, redisotel.WithMeterProvider(o.MeterProvider))
	}
	if err := redisotel.InstrumentMetrics(client, metricsOpts...); err != nil {
		return nil, fmt.Errorf("failed to instrument redis client with meter provider: %w", err)
	}

	return client, nil
}
