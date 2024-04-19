package config

import "errors"

type PostgresConfig struct {
	// URL is the PostgreSQL database connection URL.
	URL string `yaml:"url"`
}

// Validate validates the configuration.
func (c PostgresConfig) Validate() error {
	if c.URL == "" {
		return errors.New("database URL is required")
	}

	return nil
}
