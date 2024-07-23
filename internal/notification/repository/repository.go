package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/notification"
)

var _ notification.Repository = (*repository)(nil)

type repository struct {
	*postgresAdapter
	*clickhouseAdapter
}

type Config struct {
	Postgres   PostgresAdapterConfig
	Clickhouse ClickhouseAdapterConfig
}

func (c *Config) Validate() error {
	if err := c.Postgres.Validate(); err != nil {
		return err
	}

	if err := c.Clickhouse.Validate(); err != nil {
		return err
	}

	return nil
}

func New(config Config) (notification.Repository, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Initialize Postgres adapter
	pg := newPostgresAdapter(config.Postgres)

	// Initialize Clickhouse adapter
	ch := newClickhouseAdapter(config.Clickhouse)
	if err := ch.init(ctx); err != nil {
		return nil, fmt.Errorf("initializing clickhouse adapter: %w", err)
	}

	return &repository{
		postgresAdapter:   pg,
		clickhouseAdapter: ch,
	}, nil
}
