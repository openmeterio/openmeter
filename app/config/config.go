// Package config loads application configuration.
package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ReservedEventTypePattern = string

// Configuration holds any kind of Configuration that comes from the outside world and
// is necessary for running the application.
type Configuration struct {
	Address     string
	Environment string

	Telemetry TelemetryConfig

	Termination TerminationConfig

	Aggregation        AggregationConfiguration
	Entitlements       EntitlementsConfiguration
	Customer           CustomerConfiguration
	Dedupe             DedupeConfiguration
	Events             EventsConfiguration
	Ingest             IngestConfiguration
	Meters             []*meter.Meter
	ReservedEventTypes []ReservedEventTypePattern
	Namespace          NamespaceConfiguration
	Portal             PortalConfiguration
	Postgres           PostgresConfig
	Sink               SinkConfiguration
	BalanceWorker      BalanceWorkerConfiguration
	Notification       NotificationConfiguration
	ProductCatalog     ProductCatalogConfiguration
	ProgressManager    ProgressManagerConfiguration
	Billing            BillingConfiguration
	Apps               AppsConfiguration
	Svix               SvixConfig
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

	for idx, m := range c.Meters {
		// Set managed resource
		c.Meters[idx].ManagedResource = models.ManagedResource{
			NamespacedModel: models.NamespacedModel{
				Namespace: c.Namespace.Default,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			ID: m.Key,
			// Meter used to not have a name,
			// so we need to coalesce it with the key when it comes from the config
			Name:        lo.CoalesceOrEmpty(m.Name, m.Key),
			Description: m.Description,
		}

		if err := c.Meters[idx].Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	for _, pattern := range c.ReservedEventTypes {
		if _, err := regexp.Compile(pattern); err != nil {
			errs = append(errs, fmt.Errorf("reserved event type pattern %q: invalid regular expression", pattern))
		}
	}

	if err := c.BalanceWorker.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "balance worker"))
	}

	if err := c.Notification.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "notification"))
	}

	if err := c.Svix.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "svix"))
	}

	if err := c.ProductCatalog.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "product catalog"))
	}

	if err := c.Billing.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "billing"))
	}

	if err := c.Apps.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "apps"))
	}

	if err := c.ProgressManager.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "progress manager"))
	}

	if err := c.Termination.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "termination"))
	}

	if err := c.Customer.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "customer"))
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

	ConfigurePostgres(v, "postgres")
	// TODO: This is set to ensure backwards compatibility with the old config, however it should be removed in the future.
	//
	// In cloud this must never be explicitly set to prevent accidental behavior, so let's not add any kind of defaulting in
	// the reusable config parts.
	v.SetDefault("postgres.autoMigrate", "ent")

	ConfigureNamespace(v)
	ConfigureIngest(v)
	ConfigureAggregation(v)
	ConfigureSink(v)
	ConfigureDedupe(v)
	ConfigurePortal(v)
	ConfigureEvents(v)
	ConfigureBalanceWorker(v)
	ConfigureNotification(v)
	ConfigureBilling(v, flags)
	ConfigureProductCatalog(v)
	ConfigureApps(v, flags)
	ConfigureEntitlements(v, flags)
	ConfigureTermination(v, "termination")
	ConfigureProgressManager(v)
	ConfigureCustomer(v, "customer")
}
