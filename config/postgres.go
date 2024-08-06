package config

import "errors"

type PostgresConfig struct {
	// URL is the PostgreSQL database connection URL.
	URL string `yaml:"url"`
	// AutoMigrate is a flag that indicates whether the database should be automatically migrated.
	AutoMigrate bool `yaml:"autoMigrate"`
}

// Validate validates the configuration.
func (c PostgresConfig) Validate() error {
	if c.URL == "" {
		return errors.New("database URL is required")
	}

	return nil
}
