package config

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/samber/lo"
	"github.com/spf13/viper"
)

type PostgresConfig struct {
	// PostgresConnectionParams is the PostgreSQL connection parameters, URL and PostgresConnectionParams are mutually exclusive.
	PostgresConnectionParams `mapstructure:",squash"`

	// URL is the PostgreSQL database connection URL.
	URL string `yaml:"url"`

	// AutoMigrate is a flag that indicates whether the database should be automatically migrated.
	// Supported values are:
	// - "false" to disable auto-migration at startup
	// - "ent" to use ent Schema Upserts (the default value)
	// - "migration" to use the migrations directory
	AutoMigrate AutoMigrate `yaml:"autoMigrate"`
}

// Validate validates the configuration.
func (c PostgresConfig) Validate() error {
	var errs []error
	if c.URL == "" && c.PostgresConnectionParams.IsEmpty() {
		errs = append(errs, errors.New("database URL or connection params are required"))
	}

	if c.URL != "" && !c.PostgresConnectionParams.IsEmpty() {
		errs = append(errs, errors.New("database URL and connection params are mutually exclusive"))
	}

	if err := c.AutoMigrate.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (c PostgresConfig) AsURL() string {
	if c.URL != "" {
		return c.URL
	}
	return c.PostgresConnectionParams.AsURL()
}

func ConfigurePostgres(v *viper.Viper, prefix string) {
	v.SetDefault(AddPrefix(prefix, "url"), "")
	v.SetDefault(AddPrefix(prefix, "options.poolMaxConns"), 0)
	v.SetDefault(AddPrefix(prefix, "options.applicationName"), "")
	v.SetDefault(AddPrefix(prefix, "options.sslVerify"), "")
	v.SetDefault(AddPrefix(prefix, "options.sslRootCert"), "")
	v.SetDefault(AddPrefix(prefix, "host"), "")
	v.SetDefault(AddPrefix(prefix, "port"), 0)
	v.SetDefault(AddPrefix(prefix, "database"), "")
	v.SetDefault(AddPrefix(prefix, "user"), "")
	v.SetDefault(AddPrefix(prefix, "password"), "")
}

type AutoMigrate string

const (
	AutoMigrateEnt          AutoMigrate = "ent"
	AutoMigrateMigration    AutoMigrate = "migration"
	AutoMigrateMigrationJob AutoMigrate = "migration-job"
	AutoMigrateOff          AutoMigrate = "false"
)

func (a AutoMigrate) Enabled() bool {
	// For all other values it's enabled
	return a != "false"
}

func (a AutoMigrate) Validate() error {
	switch a {
	case AutoMigrateEnt, AutoMigrateMigration, AutoMigrateMigrationJob, AutoMigrateOff:
		return nil
	default:
		return errors.New("invalid auto-migrate value")
	}
}

type PostgresConnectionParams struct {
	Host     string `yaml:"host"`
	Port     uint16 `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`

	Options PostgresConnectionOptions `yaml:"options"`
}

func (c PostgresConnectionParams) Validate() error {
	var errs []error

	if c.Host == "" {
		errs = append(errs, errors.New("host is required"))
	}

	if c.Database == "" {
		errs = append(errs, errors.New("database is required"))
	}

	if c.User == "" {
		errs = append(errs, errors.New("user is required"))
	}

	if err := c.Options.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (c PostgresConnectionParams) IsEmpty() bool {
	return lo.IsEmpty(c)
}

func (c PostgresConnectionParams) AsURL() string {
	host := c.Host
	if c.Port != 0 {
		host = fmt.Sprintf("%s:%d", c.Host, c.Port)
	}

	runtimeParams := make(url.Values)

	if c.Options.ApplicationName != "" {
		runtimeParams.Set("application_name", c.Options.ApplicationName)
	}

	if c.Options.PoolMaxConns != 0 {
		runtimeParams.Set("pool_max_conns", strconv.Itoa(c.Options.PoolMaxConns))
	}

	if c.Options.SSLVerify != "" {
		runtimeParams.Set("sslmode", c.Options.SSLVerify)
	}

	if c.Options.SSLRootCert != "" {
		runtimeParams.Set("sslrootcert", c.Options.SSLRootCert)
	}

	url := url.URL{
		Scheme:   "postgresql",
		User:     url.UserPassword(c.User, c.Password),
		Host:     host,
		Path:     c.Database,
		RawQuery: runtimeParams.Encode(),
	}

	return url.String()
}

type PostgresConnectionOptions struct {
	PoolMaxConns    int    `yaml:"poolMaxConns"`
	ApplicationName string `yaml:"applicationName"`
	SSLVerify       string `yaml:"sslVerify"`
	SSLRootCert     string `yaml:"sslRootCert"`
}

func (c PostgresConnectionOptions) Validate() error {
	var errs []error

	if c.PoolMaxConns < 0 {
		errs = append(errs, errors.New("poolMaxConns must be greater than 0"))
	}

	return errors.Join(errs...)
}
