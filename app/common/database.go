package common

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

func NewPostgresDriver(
	ctx context.Context,
	conf config.PostgresConfig,
	meterProvider metric.MeterProvider,
	meter metric.Meter,
	tracerProvider trace.TracerProvider,
	logger *slog.Logger,
) (*pgdriver.Driver, func(), error) {
	driver, err := pgdriver.NewPostgresDriver(
		ctx,
		conf.URL,
		pgdriver.WithMetricMeter(meter),
		pgdriver.WithTracerProvider(tracerProvider),
		pgdriver.WithMeterProvider(meterProvider),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize postgres driver: %w", err)
	}

	return driver, func() {
		err := driver.Close()
		if err != nil {
			logger.Error("failed to close postgres driver", "error", err)
		}
	}, nil
}

// TODO: add closer function?
func NewDB(driver *pgdriver.Driver) *sql.DB {
	return driver.DB()
}

func NewEntPostgresDriver(db *sql.DB, logger *slog.Logger) (*entdriver.EntPostgresDriver, func()) {
	driver := entdriver.NewEntPostgresDriver(db)

	return driver, func() {
		err := driver.Close()
		if err != nil {
			logger.Error("failed to close ent driver", "error", err)
		}
	}
}

// TODO: add closer function?
func NewEntClient(driver *entdriver.EntPostgresDriver) *db.Client {
	return driver.Client()
}
