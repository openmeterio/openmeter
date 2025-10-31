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

	"github.com/golang-cz/devslog"
	"github.com/lmittmann/tint"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/openmeterio/openmeter/pkg/errorsx"
)

type OTLPExporterTelemetryConfig struct {
	Address string
}

// Validate validates the configuration.
func (c OTLPExporterTelemetryConfig) Validate() error {
	var errs []error

	if c.Address == "" {
		errs = append(errs, errors.New("address is required"))
	}

	return errors.Join(errs...)
}

func (c OTLPExporterTelemetryConfig) DialExporter(ctx context.Context) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		c.Address,
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("connecting to collector: %w", err)
	}

	return conn, nil
}

type AttributeSchemaType string

const (
	AttributeSchemaTypeOTel    AttributeSchemaType = "otel"
	AttributeSchemaTypeDatadog AttributeSchemaType = "datadog"
)

type TelemetryConfig struct {
	// Telemetry HTTP server address
	Address string

	AttributeSchema AttributeSchemaType

	Trace TraceTelemetryConfig

	Metrics MetricsTelemetryConfig

	Log LogTelemetryConfig
}

// Validate validates the configuration.
func (c TelemetryConfig) Validate() error {
	var errs []error

	if c.Address == "" {
		errs = append(errs, errors.New("http server address is required"))
	}

	if !slices.Contains([]AttributeSchemaType{AttributeSchemaTypeOTel, AttributeSchemaTypeDatadog}, c.AttributeSchema) {
		errs = append(errs, fmt.Errorf("invalid attribute schema: %s", c.AttributeSchema))
	}

	if err := c.Trace.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "trace"))
	}

	if err := c.Metrics.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "metrics"))
	}

	if err := c.Log.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "log"))
	}

	return errors.Join(errs...)
}

type TraceTelemetryConfig struct {
	Sampler   string
	Exporters ExportersTraceTelemetryConfig
}

// Validate validates the configuration.
func (c TraceTelemetryConfig) Validate() error {
	var errs []error

	if _, err := strconv.ParseFloat(c.Sampler, 64); err != nil && !slices.Contains([]string{"always", "never"}, c.Sampler) {
		errs = append(errs, fmt.Errorf("sampler either needs to be always|never or a ration, got: %s", c.Sampler))
	}

	if err := c.Exporters.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "exporters"))
	}

	return errors.Join(errs...)
}

type ExportersTraceTelemetryConfig struct {
	OTLP    OTLPExportersTraceTelemetryConfig
	DataDog DataDogExportersTraceTelemetryConfig
}

// Validate validates the configuration.
func (c ExportersTraceTelemetryConfig) Validate() error {
	var errs []error

	if err := c.OTLP.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "otlp"))
	}

	if err := c.DataDog.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "datadog"))
	}

	if c.OTLP.Enabled && c.DataDog.Enabled {
		errs = append(errs, errors.New("only one exporter can be enabled (oltp vs datadog)"))
	}

	return errors.Join(errs...)
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

type DataDogExportersTraceTelemetryConfig struct {
	Enabled bool
	Debug   bool
}

// Validate validates the configuration.
func (c DataDogExportersTraceTelemetryConfig) Validate() error {
	return nil
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
	var errs []error

	if err := c.Prometheus.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "prometheus"))
	}

	if err := c.OTLP.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "otlp"))
	}

	return errors.Join(errs...)
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

type LogTelemetryConfig struct {
	// Format specifies the output log format.
	// Accepted values are: json, text
	Format string

	// Level is the minimum log level that should appear on the output.
	//
	// Requires [mapstructure.TextUnmarshallerHookFunc] to be high up in the decode hook chain.
	Level slog.Level

	Exporters ExportersLogTelemetryConfig
}

// Validate validates the configuration.
func (c LogTelemetryConfig) Validate() error {
	var errs []error

	if !slices.Contains([]string{"json", "text", "tint", "prettydev"}, c.Format) {
		errs = append(errs, fmt.Errorf("invalid format: %q", c.Format))
	}

	if !slices.Contains([]slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}, c.Level) {
		errs = append(errs, fmt.Errorf("invalid level: %q", c.Level))
	}

	return errors.Join(errs...)
}

// NewHandler creates a new [slog.Handler].
func (c LogTelemetryConfig) NewHandler(w io.Writer) slog.Handler {
	switch c.Format {
	case "json":
		return slog.NewJSONHandler(w, &slog.HandlerOptions{Level: c.Level})

	case "text":
		return slog.NewTextHandler(w, &slog.HandlerOptions{Level: c.Level})

	case "tint":
		return tint.NewHandler(os.Stdout, &tint.Options{Level: c.Level})
	case "prettydev":
		return devslog.NewHandler(w, &devslog.Options{
			MaxSlicePrintSize: 4,
			SortKeys:          true,
			TimeFormat:        "[04:05]",
			NewLineAfterLog:   true,
			DebugColor:        devslog.Magenta,
			HandlerOptions:    &slog.HandlerOptions{Level: c.Level},
		})
	}

	return slog.NewJSONHandler(w, &slog.HandlerOptions{Level: c.Level})
}

type ExportersLogTelemetryConfig struct {
	OTLP   OTLPExportersLogTelemetryConfig
	Stdout StdoutExportersLogTelemetryConfig
	File   FileExportersLogTelemetryConfig
}

// Validate validates the configuration.
func (c ExportersLogTelemetryConfig) Validate() error {
	var errs []error

	if err := c.OTLP.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "otlp"))
	}

	if err := c.File.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "file"))
	}

	return errors.Join(errs...)
}

type OTLPExportersLogTelemetryConfig struct {
	Enabled bool

	OTLPExporterTelemetryConfig `mapstructure:",squash"`
}

// Validate validates the configuration.
func (c OTLPExportersLogTelemetryConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	return c.OTLPExporterTelemetryConfig.Validate()
}

// NewExporter creates a new [sdklog.Exporter].
func (c OTLPExportersLogTelemetryConfig) NewExporter(ctx context.Context) (sdklog.Exporter, error) {
	if !c.Enabled {
		return nil, errors.New("telemetry: log: exporter: otlp: disabled")
	}

	// TODO: make this configurable
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	conn, err := c.DialExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("telemetry: log: exporter: otlp: %w", err)
	}

	exporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("telemetry: log: exporter: otlp: initializing exporter: %w", err)
	}

	return exporter, nil
}

// StdoutExportersLogTelemetryConfig represents the configuration for the stdout log exporter.
// See https://pkg.go.dev/go.opentelemetry.io/otel/exporters/stdout/stdoutlog
type StdoutExportersLogTelemetryConfig struct {
	Enabled     bool
	PrettyPrint bool
}

// Validate validates the configuration.
func (c StdoutExportersLogTelemetryConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	return nil
}

// NewExporter creates a new [sdklog.Exporter].
func (c StdoutExportersLogTelemetryConfig) NewExporter() (sdklog.Exporter, error) {
	if !c.Enabled {
		return nil, errors.New("telemetry: log: exporter: stdout: disabled")
	}

	var opts []stdoutlog.Option

	if c.PrettyPrint {
		opts = append(opts, stdoutlog.WithPrettyPrint())
	}

	return stdoutlog.New(opts...)
}

// FileExportersLogTelemetryConfig represents the configuration for the file log exporter.
type FileExportersLogTelemetryConfig struct {
	Enabled     bool
	FilePath    string
	PrettyPrint bool
}

// Validate validates the configuration.
func (c FileExportersLogTelemetryConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.FilePath == "" {
		return errors.New("file path is required")
	}

	return nil
}

// fileExporter wraps an slog.Handler as an OpenTelemetry exporter
type fileExporter struct {
	handler slog.Handler
	file    *os.File
}

func (e *fileExporter) Export(ctx context.Context, logs []sdklog.Record) error {
	for _, record := range logs {
		// Convert OTel severity to slog level
		level := slog.LevelInfo
		switch record.Severity() {
		case log.SeverityTrace, log.SeverityDebug:
			level = slog.LevelDebug
		case log.SeverityInfo:
			level = slog.LevelInfo
		case log.SeverityWarn:
			level = slog.LevelWarn
		case log.SeverityError, log.SeverityFatal:
			level = slog.LevelError
		}

		attrs := make([]any, 0, record.AttributesLen()*2)
		record.WalkAttributes(func(attr log.KeyValue) bool {
			attrs = append(attrs, attr.Key, attr.Value.AsString())
			return true
		})

		rec := slog.NewRecord(
			record.Timestamp(),
			level,
			record.Body().AsString(),
			0,
		)

		rec.Add(attrs...)

		// Let's add retries eventually
		_ = e.handler.Handle(ctx, rec)
	}
	return nil
}

func (e *fileExporter) ForceFlush(ctx context.Context) error {
	return e.file.Sync()
}

func (e *fileExporter) Shutdown(ctx context.Context) error {
	return e.file.Close()
}

// NewExporter creates a new [sdklog.Exporter].
func (c FileExportersLogTelemetryConfig) NewExporter() (sdklog.Exporter, error) {
	if !c.Enabled {
		return nil, errors.New("telemetry: log: exporter: file: disabled")
	}

	f, err := os.OpenFile(c.FilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("telemetry: log: exporter: file: opening file: %w", err)
	}

	handler := slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	return &fileExporter{
		handler: handler,
		file:    f,
	}, nil
}

func (c LogTelemetryConfig) NewLoggerProvider(ctx context.Context, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	options := []sdklog.LoggerProviderOption{
		sdklog.WithResource(res),
	}

	if c.Exporters.OTLP.Enabled {
		exporter, err := c.Exporters.OTLP.NewExporter(ctx)
		if err != nil {
			return nil, fmt.Errorf("exporter: otlp: %w", err)
		}

		options = append(options, sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)))
	}

	if c.Exporters.Stdout.Enabled {
		exporter, err := c.Exporters.Stdout.NewExporter()
		if err != nil {
			return nil, fmt.Errorf("exporter: stdout: %w", err)
		}

		options = append(options, sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)))
	}

	if c.Exporters.File.Enabled {
		exporter, err := c.Exporters.File.NewExporter()
		if err != nil {
			return nil, fmt.Errorf("exporter: file: %w", err)
		}

		// Use simple processor for immediate writes to file
		options = append(options, sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)))
	}

	return sdklog.NewLoggerProvider(options...), nil
}

// ConfigureTelemetry configures some defaults in the Viper instance.
func ConfigureTelemetry(v *viper.Viper, flags *pflag.FlagSet) {
	flags.String("telemetry-address", ":10000", "Telemetry HTTP server address")
	_ = v.BindPFlag("telemetry.address", flags.Lookup("telemetry-address"))
	v.SetDefault("telemetry.address", ":10000")

	v.SetDefault("telemetry.attributeSchema", AttributeSchemaTypeOTel)

	v.SetDefault("telemetry.trace.sampler", "never")
	v.SetDefault("telemetry.trace.exporters.otlp.enabled", false)
	v.SetDefault("telemetry.trace.exporters.otlp.address", "")
	v.SetDefault("telemetry.trace.exporters.datadog.enabled", false)
	v.SetDefault("telemetry.trace.exporters.datadog.debug", false)

	v.SetDefault("telemetry.metrics.exporters.prometheus.enabled", false)
	v.SetDefault("telemetry.metrics.exporters.otlp.enabled", false)
	v.SetDefault("telemetry.metrics.exporters.otlp.address", "")

	v.SetDefault("telemetry.log.format", "json")
	v.SetDefault("telemetry.log.level", "info")
	v.SetDefault("telemetry.log.exporters.otlp.enabled", false)
	v.SetDefault("telemetry.log.exporters.otlp.address", "")
	v.SetDefault("telemetry.log.exporters.stdout.enabled", false)
	v.SetDefault("telemetry.log.exporters.file.enabled", false)
	v.SetDefault("telemetry.log.exporters.file.filepath", "")
	v.SetDefault("telemetry.log.exporters.file.prettyprint", false)

	v.SetDefault("telemetry.readiness.interval", 3*time.Second)
}
