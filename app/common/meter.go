package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4/database/clickhouse"
	"github.com/google/wire"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/app/config"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meter/adapter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/registry"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

var MeterInMemory = wire.NewSet(
	wire.FieldsOf(new(config.Configuration), "Meters"),

	NewMeterService,
)

func NewMeterService(
	logger *slog.Logger,
	db *entdb.Client,
) (meter.Service, error) {
	service, err := adapter.New(adapter.Config{
		Client: db,
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}

	return service, nil
}

func NewMeterManageService(
	ctx context.Context,
	db *entdb.Client,
	logger *slog.Logger,
	entitlementRegistry *registry.Entitlement,
	clickHouse clickhouse.ClickHouse,
	namespaceManager *namespace.Manager,
	streamingConnector streaming.Connector,
	configMeters []*meter.Meter,
) (meter.ManageService, error) {
	meterManageService, err := adapter.NewManage(adapter.ManageConfig{
		Config: adapter.Config{
			Client: db,
			Logger: logger,
		},
		EntitlementRepository: entitlementRegistry.EntitlementRepo,
		FeatureRepository:     entitlementRegistry.FeatureRepo,
		NamespaceManager:      namespaceManager,
		StreamingConnector:    streamingConnector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create meter manage service: %w", err)
	}

	// Create config meters if they don't exist in the database
	if len(configMeters) > 0 {
		createConfigMetersInDatabase(ctx, logger, configMeters, namespaceManager, meterManageService)
	}

	return meterManageService, nil
}

// createConfigMetersInDatabase creates meters in the database if they don't exist
func createConfigMetersInDatabase(
	ctx context.Context,
	logger *slog.Logger,
	configMeters []*meter.Meter,
	namespaceManager *namespace.Manager,
	meterService meter.ManageService,
) error {
	configMeterSlugs := lo.Map(configMeters, func(meter *meter.Meter, _ int) string {
		return meter.Slug
	})

	meters, err := meter.ListAll(ctx, meterService, meter.ListMetersParams{
		Namespace:  namespaceManager.GetDefaultNamespace(),
		SlugFilter: &configMeterSlugs,
	})
	if err != nil {
		return fmt.Errorf("failed to list meters: %w", err)
	}

	metersBySlug := lo.KeyBy(meters, func(meter meter.Meter) string {
		return meter.Slug
	})

	for _, configMeter := range configMeters {
		configMeter.Namespace = namespaceManager.GetDefaultNamespace()

		// Backfill the name if it's missing
		if configMeter.Name == "" {
			configMeter.Name = configMeter.Slug
		}

		// Create the meter if it doesn't exist
		if dbMeter, ok := metersBySlug[configMeter.Slug]; !ok {
			_, err := meterService.CreateMeter(ctx, meter.CreateMeterInput{
				Namespace:     configMeter.Namespace,
				Key:           configMeter.Slug,
				Name:          configMeter.Name,
				Description:   configMeter.Description,
				Aggregation:   configMeter.Aggregation,
				EventType:     configMeter.EventType,
				ValueProperty: configMeter.ValueProperty,
				GroupBy:       configMeter.GroupBy,
			})
			if err != nil {
				return fmt.Errorf("failed to create meter: %w", err)
			}

			logger.InfoContext(ctx, "created meter in database", "meter", configMeter.Slug)
		} else {
			if !dbMeter.Equal(*configMeter) {
				return fmt.Errorf("meter %s in database is not equal to the meter in config", dbMeter.Slug)
			}

			logger.InfoContext(ctx, "meter in config already exists in database", "meter", configMeter.Slug)
		}
	}

	return nil
}
