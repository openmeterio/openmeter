// Package config loads application configuration.
package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Configuration holds any kind of Configuration that comes from the outside world and
// is necessary for running the application.
type Configuration struct {
	Address     string
	Environment string

	Telemetry TelemetryConfig

	Aggregation         AggregationConfiguration
	Entitlements        EntitlementsConfiguration
	Dedupe              DedupeConfiguration
	Events              EventsConfiguration
	Ingest              IngestConfiguration
	Meters              []*models.Meter
	Namespace           NamespaceConfiguration
	Portal              PortalConfiguration
	Postgres            PostgresConfig
	Sink                SinkConfiguration
	BalanceWorker       BalanceWorkerConfiguration
	NotificationService NotificationServiceConfiguration
	Svix                notificationwebhook.SvixConfig
}

// Validate validates the configuration.
func (c Configuration) Validate() error {
	if c.Address == "" {
		return errors.New("server address is required")
	}

	if err := c.Telemetry.Validate(); err != nil {
		return fmt.Errorf("telemetry: %w", err)
	}

	if err := c.Namespace.Validate(); err != nil {
		return fmt.Errorf("namespace: %w", err)
	}

	if err := c.Ingest.Validate(); err != nil {
		return fmt.Errorf("ingest: %w", err)
	}

	if err := c.Aggregation.Validate(); err != nil {
		return fmt.Errorf("aggregation: %w", err)
	}

	if err := c.Sink.Validate(); err != nil {
		return fmt.Errorf("sink: %w", err)
	}

	if err := c.Dedupe.Validate(); err != nil {
		return fmt.Errorf("dedupe: %w", err)
	}

	if err := c.Portal.Validate(); err != nil {
		return fmt.Errorf("portal: %w", err)
	}

	if err := c.Entitlements.Validate(); err != nil {
		return fmt.Errorf("entitlements: %w", err)
	}

	if len(c.Meters) == 0 {
		return errors.New("no meters configured: add meter to configuration file")
	}

	for _, m := range c.Meters {
		// Namespace is not configurable on per meter level
		m.Namespace = c.Namespace.Default

		// set default window size
		if m.WindowSize == "" {
			m.WindowSize = models.WindowSizeMinute
		}

		if err := m.Validate(); err != nil {
			return err
		}
	}

	if err := c.BalanceWorker.Validate(); err != nil {
		return fmt.Errorf("balance worker: %w", err)
	}

	if err := c.NotificationService.Validate(); err != nil {
		return fmt.Errorf("notification service: %w", err)
	}

	if err := c.Svix.Validate(); err != nil {
		return fmt.Errorf("svix: %w", err)
	}

	return nil
}

func SetViperDefaults(v *viper.Viper, flags *pflag.FlagSet) {
	// Viper settings
	// TODO: remove this: it's not in use
	v.AddConfigPath(".")

	// Environment variable settings
	// TODO: replace this with constructor option
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	// Server configuration
	flags.String("address", ":8888", "Server address")
	_ = v.BindPFlag("address", flags.Lookup("address"))
	v.SetDefault("address", ":8888")

	// Environment used for identifying the service environment
	v.SetDefault("environment", "unknown")

	configureTelemetry(v, flags)

	ConfigureNamespace(v)
	ConfigureIngest(v)
	ConfigureAggregation(v)
	ConfigureSink(v)
	ConfigureDedupe(v)
	ConfigurePortal(v)
	ConfigureEvents(v)
	ConfigureBalanceWorker(v)
	ConfigureNotificationService(v)
}
