package entitlement

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/internal/ent/db"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/registry"
	registrybuilder "github.com/openmeterio/openmeter/internal/registry/builder"
	"github.com/openmeterio/openmeter/internal/streaming/clickhouse_connector"
	watermillkafka "github.com/openmeterio/openmeter/internal/watermill/driver/kafka"
	"github.com/openmeterio/openmeter/internal/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type entitlementConnectors struct {
	Registry *registry.Entitlement
	EventBus eventbus.Publisher
	Shutdown func()
}

func initEntitlements(ctx context.Context, conf config.Configuration, logger *slog.Logger, metricMeter metric.Meter, otelName string) (*entitlementConnectors, error) {
	// Postgresql
	entDriver, err := entutils.GetPGDriver(conf.Postgres.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to init postgres driver: %w", err)
	}

	dbClient := db.NewClient(db.Driver(entDriver))

	// Meter repository
	meterRepository := meter.NewInMemoryRepository(slicesx.Map(conf.Meters, func(meter *models.Meter) models.Meter {
		return *meter
	}))

	// streaming connector
	clickHouseClient, err := clickhouse.Open(conf.Aggregation.ClickHouse.GetClientOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize clickhouse client: %w", err)
	}

	streamingConnector, err := clickhouse_connector.NewClickhouseConnector(clickhouse_connector.ClickhouseConnectorConfig{
		Logger:               logger,
		ClickHouse:           clickHouseClient,
		Database:             conf.Aggregation.ClickHouse.Database,
		Meters:               meterRepository,
		CreateOrReplaceMeter: conf.Aggregation.CreateOrReplaceMeter,
		PopulateMeter:        conf.Aggregation.PopulateMeter,
	})
	if err != nil {
		return nil, fmt.Errorf("init clickhouse streaming: %w", err)
	}

	// event publishing
	eventPublisherDriver, err := watermillkafka.NewPublisher(ctx, watermillkafka.PublisherOptions{
		Broker: watermillkafka.BrokerOptions{
			KafkaConfig:  conf.Ingest.Kafka.KafkaConfiguration,
			ClientID:     otelName,
			Logger:       logger,
			MetricMeter:  metricMeter,
			DebugLogging: conf.Telemetry.Log.Level == slog.LevelDebug,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize event publisher driver: %w", err)
	}

	eventPublisher, err := eventbus.New(eventbus.Options{
		Publisher:              eventPublisherDriver,
		Config:                 conf.Events,
		Logger:                 logger,
		MarshalerTransformFunc: watermillkafka.AddPartitionKeyFromSubject,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize event publisher: %w", err)
	}

	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registry.EntitlementOptions{
		DatabaseClient:     dbClient,
		StreamingConnector: streamingConnector,
		MeterRepository:    meterRepository,
		Logger:             logger,
		Publisher:          eventPublisher,
	})

	return &entitlementConnectors{
		Registry: entitlementRegistry,
		EventBus: eventPublisher,
		Shutdown: func() {
			if err := dbClient.Close(); err != nil {
				logger.Error("failed to close entitlement db client", "error", err)
			}

			if err := clickHouseClient.Close(); err != nil {
				logger.Error("failed to close clickhouse client", "error", err)
			}

			if err := eventPublisherDriver.Close(); err != nil {
				logger.Error("failed to close event publisher", "error", err)
			}
		},
	}, nil
}
