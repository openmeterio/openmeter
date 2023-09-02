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
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TelemetryConfig struct {
	// Telemetry HTTP server address
	Address string

	Trace TraceTelemetryConfig

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

	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log: %w", err)
	}

	return nil
}

type TraceTelemetryConfig struct {
	Exporter ExporterTraceTelemetryConfig
	Sampler  string
}

// Validate validates the configuration.
func (c TraceTelemetryConfig) Validate() error {
	if _, err := strconv.ParseFloat(c.Sampler, 64); err != nil && !slices.Contains([]string{"always", "never"}, c.Sampler) {
		return fmt.Errorf("sampler either needs to be always|never or a ration, got: %s", c.Sampler)
	}

	if err := c.Exporter.Validate(); err != nil {
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

type ExporterTraceTelemetryConfig struct {
	Enabled bool
	Address string
}

// Validate validates the configuration.
func (c ExporterTraceTelemetryConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Address == "" {
		return errors.New("address is required")
	}

	return nil
}

func (c ExporterTraceTelemetryConfig) GetExporter() (sdktrace.SpanExporter, error) {
	if !c.Enabled {
		return nil, errors.New("telemetry: trace: exporter: disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		c.Address,
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry: trace: exporter: %w", err)
	}

	exporter, err := otlptracegrpc.New(context.Background(), otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("telemetry: trace: exporter: failed to create: %w", err)
	}

	return exporter, nil
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

// ConfigureTelemetry configures some defaults in the Viper instance.
func ConfigureTelemetry(v *viper.Viper, flags *pflag.FlagSet) {
	flags.String("telemetry-address", ":10000", "Telemetry HTTP server address")
	_ = v.BindPFlag("telemetry.address", flags.Lookup("telemetry-address"))
	v.SetDefault("telemetry.address", ":10000")

	v.SetDefault("telemetry.trace.sampler", "never")
	v.SetDefault("telemetry.trace.exporter.enabled", false)
	v.SetDefault("telemetry.trace.exporter.address", "")

	v.SetDefault("telemetry.log.format", "json")
	v.SetDefault("telemetry.log.level", "info")
}
