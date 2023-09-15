package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/lmittmann/tint"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type OTLPExporterTelemetryConfig struct {
	Address string
}

// Validate validates the configuration.
func (c OTLPExporterTelemetryConfig) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}

	return nil
}

func (c OTLPExporterTelemetryConfig) DialExporter(ctx context.Context) (*grpc.ClientConn, error) {
	conn, err := grpc.DialContext(
		ctx,
		c.Address,
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("connecting to collector: %w", err)
	}

	return conn, nil
}

type TelemetryConfig struct {
	// Telemetry HTTP server address
	Address string

	Trace TraceTelemetryConfig

	Metrics MetricsTelemetryConfig

	Log LogTelemetryConfiguration
}

// Validate validates the configuration.
func (c TelemetryConfig) Validate() error {
	if c.Address == "" {
		return errors.New("http server address is required")
	}

	if err := c.Trace.Validate(); err != nil {
		return fmt.Errorf("trace: %w", err)
	}

	if err := c.Metrics.Validate(); err != nil {
		return fmt.Errorf("metrics: %w", err)
	}

	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log: %w", err)
	}

	return nil
}

type TraceTelemetryConfig struct {
	Sampler   string
	Exporters ExportersTraceTelemetryConfig
}

// Validate validates the configuration.
func (c TraceTelemetryConfig) Validate() error {
	if _, err := strconv.ParseFloat(c.Sampler, 64); err != nil && !slices.Contains([]string{"always", "never"}, c.Sampler) {
		return fmt.Errorf("sampler either needs to be always|never or a ration, got: %s", c.Sampler)
	}

	if err := c.Exporters.Validate(); err != nil {
		return fmt.Errorf("exporter: %w", err)
	}

	return nil
}

func (c TraceTelemetryConfig) GetSampler() sdktrace.Sampler {
	switch c.Sampler {
	case "always":
		return sdktrace.AlwaysSample()

	case "never":
		return sdktrace.NeverSample()

	default:
		ratio, err := strconv.ParseFloat(c.Sampler, 64)
		if err != nil {
			panic(fmt.Errorf("trace: invalid ratio: %w", err))
		}

		return sdktrace.TraceIDRatioBased(ratio)
	}
}

func (c TraceTelemetryConfig) NewTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	options := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(c.GetSampler()),
	}

	if c.Exporters.OTLP.Enabled {
		exporter, err := c.Exporters.OTLP.NewExporter(ctx)
		if err != nil {
			return nil, err
		}

		options = append(options, sdktrace.WithBatcher(exporter))
	}

	return sdktrace.NewTracerProvider(options...), nil
}

type ExportersTraceTelemetryConfig struct {
	OTLP OTLPExportersTraceTelemetryConfig
}

// Validate validates the configuration.
func (c ExportersTraceTelemetryConfig) Validate() error {
	if err := c.OTLP.Validate(); err != nil {
		return fmt.Errorf("otlp: %w", err)
	}

	return nil
}

type OTLPExportersTraceTelemetryConfig struct {
	Enabled bool

	OTLPExporterTelemetryConfig `mapstructure:",squash"`
}

// Validate validates the configuration.
func (c OTLPExportersTraceTelemetryConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	return c.OTLPExporterTelemetryConfig.Validate()
}

// NewExporter creates a new [sdktrace.SpanExporter].
func (c OTLPExportersTraceTelemetryConfig) NewExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	if !c.Enabled {
		return nil, errors.New("telemetry: trace: exporter: otlp: disabled")
	}

	// TODO: make this configurable
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	conn, err := c.DialExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("telemetry: trace: exporter: otlp: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("telemetry: trace: exporter: otlp: initializing exporter: %w", err)
	}

	return exporter, nil
}

type MetricsTelemetryConfig struct {
	Exporters ExportersMetricsTelemetryConfig
}

// Validate validates the configuration.
func (c MetricsTelemetryConfig) Validate() error {
	if err := c.Exporters.Validate(); err != nil {
		return fmt.Errorf("exporter: %w", err)
	}

	return nil
}

func (c MetricsTelemetryConfig) NewMeterProvider(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	options := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}

	if c.Exporters.Prometheus.Enabled {
		exporter, err := c.Exporters.Prometheus.NewExporter()
		if err != nil {
			return nil, err
		}

		options = append(options, sdkmetric.WithReader(exporter))
	}

	if c.Exporters.OTLP.Enabled {
		exporter, err := c.Exporters.OTLP.NewExporter(ctx)
		if err != nil {
			return nil, err
		}

		options = append(options, sdkmetric.WithReader(exporter))
	}

	return sdkmetric.NewMeterProvider(options...), nil
}

type ExportersMetricsTelemetryConfig struct {
	Prometheus PrometheusExportersMetricsTelemetryConfig
	OTLP       OTLPExportersMetricsTelemetryConfig
}

// Validate validates the configuration.
func (c ExportersMetricsTelemetryConfig) Validate() error {
	if err := c.Prometheus.Validate(); err != nil {
		return fmt.Errorf("prometheus: %w", err)
	}

	if err := c.OTLP.Validate(); err != nil {
		return fmt.Errorf("otlp: %w", err)
	}

	return nil
}

type PrometheusExportersMetricsTelemetryConfig struct {
	Enabled bool
}

// Validate validates the configuration.
func (c PrometheusExportersMetricsTelemetryConfig) Validate() error {
	return nil
}

// NewExporter creates a new [sdkmetric.Reader].
func (c PrometheusExportersMetricsTelemetryConfig) NewExporter() (sdkmetric.Reader, error) {
	if !c.Enabled {
		return nil, errors.New("telemetry: metrics: exporter: prometheus: disabled")
	}

	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("telemetry: metrics: exporter: prometheus: initializing exporter: %w", err)
	}

	return exporter, nil
}

type OTLPExportersMetricsTelemetryConfig struct {
	Enabled bool

	OTLPExporterTelemetryConfig `mapstructure:",squash"`
}

// Validate validates the configuration.
func (c OTLPExportersMetricsTelemetryConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	return c.OTLPExporterTelemetryConfig.Validate()
}

// NewExporter creates a new [sdkmetric.Reader].
func (c OTLPExportersMetricsTelemetryConfig) NewExporter(ctx context.Context) (sdkmetric.Reader, error) {
	if !c.Enabled {
		return nil, errors.New("telemetry: metrics: exporter: otlp: disabled")
	}

	// TODO: make this configurable
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	conn, err := c.DialExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("telemetry: metrics: exporter: otlp: %w", err)
	}

	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("telemetry: metrics: exporter: otlp: initializing exporter: %w", err)
	}

	return sdkmetric.NewPeriodicReader(exporter), nil
}

type LogTelemetryConfiguration struct {
	// Format specifies the output log format.
	// Accepted values are: json, text
	Format string

	// Level is the minimum log level that should appear on the output.
	//
	// Requires [mapstructure.TextUnmarshallerHookFunc] to be high up in the decode hook chain.
	Level slog.Level
}

// Validate validates the configuration.
func (c LogTelemetryConfiguration) Validate() error {
	if !slices.Contains([]string{"json", "text", "tint"}, c.Format) {
		return fmt.Errorf("invalid format: %q", c.Format)
	}

	if !slices.Contains([]slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}, c.Level) {
		return fmt.Errorf("invalid level: %q", c.Level)
	}

	return nil
}

// NewHandler creates a new [slog.Handler].
func (c LogTelemetryConfiguration) NewHandler(w io.Writer) slog.Handler {
	switch c.Format {
	case "json":
		return slog.NewJSONHandler(w, &slog.HandlerOptions{Level: c.Level})

	case "text":
		return slog.NewTextHandler(w, &slog.HandlerOptions{Level: c.Level})

	case "tint":
		return tint.NewHandler(os.Stdout, &tint.Options{Level: c.Level})
	}

	return slog.NewJSONHandler(w, &slog.HandlerOptions{Level: c.Level})
}

// configureTelemetry configures some defaults in the Viper instance.
func configureTelemetry(v *viper.Viper, flags *pflag.FlagSet) {
	flags.String("telemetry-address", ":10000", "Telemetry HTTP server address")
	_ = v.BindPFlag("telemetry.address", flags.Lookup("telemetry-address"))
	v.SetDefault("telemetry.address", ":10000")

	v.SetDefault("telemetry.trace.sampler", "never")
	v.SetDefault("telemetry.trace.exporters.otlp.enabled", false)
	v.SetDefault("telemetry.trace.exporters.otlp.address", "")

	v.SetDefault("telemetry.metrics.exporters.prometheus.enabled", false)
	v.SetDefault("telemetry.metrics.exporters.otlp.enabled", false)
	v.SetDefault("telemetry.metrics.exporters.otlp.address", "")

	v.SetDefault("telemetry.log.format", "json")
	v.SetDefault("telemetry.log.level", "info")
}
