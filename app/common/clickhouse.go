package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/pkg/framework/clickhouseotel"
	"github.com/openmeterio/openmeter/tools/migrate"
)

var ClickHouse = wire.NewSet(
	NewClickHouse,
	wire.Struct(new(ClickHouseMigrator), "*"),
)

// ClickHouseMigrator executes ClickHouse schema migrations.
type ClickHouseMigrator struct {
	Config config.ClickHouseAggregationConfiguration
	Logger *slog.Logger
}

func (m ClickHouseMigrator) Migrate(ctx context.Context) error {
	if !m.Config.AutoMigrate.Enabled() {
		m.Logger.Debug("clickhouse auto migration is disabled")
		return nil
	}

	m.Logger.Debug("running clickhouse migrations", slog.String("strategy", string(m.Config.AutoMigrate)))

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: m.Config.AsMigrateURL(),
		Migrations:       migrate.ClickHouseMigrationsConfig,
		Logger:           m.Logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create clickhouse migrator: %w", err)
	}
	defer migrator.CloseOrLogError()

	switch m.Config.AutoMigrate {
	case config.ClickHouseAutoMigrateMigration:
		if err := migrator.Up(); err != nil {
			return fmt.Errorf("failed to migrate clickhouse: %w", err)
		}
	case config.ClickHouseAutoMigrateMigrationJob:
		if err := migrator.WaitForMigrationJob(); err != nil {
			return fmt.Errorf("failed to wait for clickhouse migration job: %w", err)
		}
	}

	m.Logger.Info("clickhouse database initialized")

	return nil
}

func NewClickHouse(ctx context.Context, conf config.ClickHouseAggregationConfiguration, tracer trace.Tracer, meter metric.Meter, logger *slog.Logger) (clickhouse.Conn, func(), error) {
	noopClose := func() {}

	closers := []func(){}

	conn, err := clickhouse.Open(conf.GetClientOptions())
	if err != nil {
		return nil, noopClose, fmt.Errorf("failed to initialize clickhouse client: %w", err)
	}

	closers = append(closers, func() {
		if err := conn.Close(); err != nil {
			logger.Error("failed to close clickhouse client", "error", err)
		}
	})

	if conf.Tracing {
		conn, err = clickhouseotel.NewClickHouseTracer(clickhouseotel.ClickHouseTracerConfig{
			Conn:   conn,
			Tracer: tracer,
		})
		if err != nil {
			return nil, noopClose, fmt.Errorf("failed to initialize clickhouse tracer: %w", err)
		}
	}

	if conf.PoolMetrics.Enabled {
		connPoolMetrics, err := clickhouseotel.NewConnPoolMetrics(clickhouseotel.ConnPoolMetricsConfig{
			Conn:         conn,
			Meter:        meter,
			PollInterval: conf.PoolMetrics.PollInterval,
			Logger:       logger,
		})
		if err != nil {
			return nil, noopClose, fmt.Errorf("failed to initialize clickhouse pool metrics: %w", err)
		}

		if err := connPoolMetrics.Start(ctx); err != nil {
			return nil, noopClose, fmt.Errorf("failed to start clickhouse pool metrics: %w", err)
		}

		closers = append(closers, func() {
			if err := connPoolMetrics.Shutdown(); err != nil {
				logger.Error("failed to shutdown clickhouse pool metrics", "error", err)
			}
		})
	}

	return conn, func() {
		for _, closer := range closers {
			closer()
		}
	}, nil
}
