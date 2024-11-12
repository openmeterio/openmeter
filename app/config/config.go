// Package config loads application configuration.
package config

import (
	"errors"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Configuration holds any kind of Configuration that comes from the outside world and
// is necessary for running the application.
type Configuration struct {
	Address     string
	Environment string

	Telemetry TelemetryConfig

	Aggregation    AggregationConfiguration
	Entitlements   EntitlementsConfiguration
	Dedupe         DedupeConfiguration
	Events         EventsConfiguration
	Ingest         IngestConfiguration
	Meters         []*models.Meter
	Namespace      NamespaceConfiguration
	Portal         PortalConfiguration
	Postgres       PostgresConfig
	Sink           SinkConfiguration
	BalanceWorker  BalanceWorkerConfiguration
	Notification   NotificationConfiguration
	ProductCatalog ProductCatalogConfiguration
	Billing        BillingConfiguration
	Apps           AppsConfiguration
	StripeApp      StripeAppConfig
	Svix           SvixConfig
}

// Validate validates the configuration.
func (c Configuration) Validate() error {
	var errs []error

	if c.Address == "" {
		errs = append(errs, errors.New("server address is required"))
	}

	if err := c.Telemetry.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "telemetry"))
	}

	if err := c.Namespace.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "namespace"))
	}

	if err := c.Ingest.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "ingest"))
	}

	if err := c.Aggregation.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "aggregation"))
	}

	if err := c.Sink.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "sink"))
	}

	if err := c.Dedupe.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "dedupe"))
	}

	if err := c.Portal.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "portal"))
	}

	if err := c.Entitlements.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "entitlements"))
	}

	if len(c.Meters) == 0 {
		errs = append(errs, errors.New("no meters configured: add meter to configuration file"))
	}

	for _, m := range c.Meters {
		// Namespace is not configurable on per meter level
		m.Namespace = c.Namespace.Default

		// set default window size
		if m.WindowSize == "" {
			m.WindowSize = models.WindowSizeMinute
		}

		if err := m.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if err := c.BalanceWorker.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "balance worker"))
	}

	if c.Notification.Enabled {
		if err := c.Notification.Validate(); err != nil {
			errs = append(errs, errorsx.WithPrefix(err, "notification"))
		}

		if err := c.Svix.Validate(); err != nil {
			errs = append(errs, errorsx.WithPrefix(err, "svix"))
		}
	}

	if err := c.StripeApp.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "stripe app"))
	}

	if err := c.ProductCatalog.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "product catalog"))
	}

	if c.ProductCatalog.Enabled && !c.Entitlements.Enabled {
		errs = append(errs, errors.New("entitlements must be enabled if product catalog is enabled"))
	}

	if err := c.Billing.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "billing"))
	}

	if err := c.Apps.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "apps"))
	}

	return errors.Join(errs...)
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

	ConfigureTelemetry(v, flags)

	ConfigurePostgres(v)
	ConfigureNamespace(v)
	ConfigureIngest(v)
	ConfigureAggregation(v)
	ConfigureSink(v)
	ConfigureDedupe(v)
	ConfigurePortal(v)
	ConfigureEvents(v)
	ConfigureBalanceWorker(v)
	ConfigureNotification(v)
	ConfigureStripe(v)
	ConfigureBilling(v)
	ConfigureProductCatalog(v)
	ConfigureApps(v)
}
