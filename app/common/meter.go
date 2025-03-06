package common

import (
	"context"
	"fmt"
	"log/slog"

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

type MeterConfigInitializer = func(ctx context.Context) error

var Meter = wire.NewSet(
	NewMeterService,
)

var MeterManage = wire.NewSet(
	Meter,
	NewMeterManageService,
)

var MeterManageWithConfigMeters = wire.NewSet(
	wire.FieldsOf(new(config.Configuration), "Meters"),

	Meter,
	NewMeterManageService,
	NewMeterConfigInitializer,
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
	namespaceManager *namespace.Manager,
	streamingConnector streaming.Connector,
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

	return meterManageService, nil
}

func NewMeterConfigInitializer(
	logger *slog.Logger,
	configMeters []*meter.Meter,
	meterManagerService meter.ManageService,
	namespaceManager *namespace.Manager,
) MeterConfigInitializer {
	return func(ctx context.Context) error {
		// Create config meters if they don't exist in the database
		if len(configMeters) > 0 {
			err := createConfigMetersInDatabase(ctx, logger, configMeters, namespaceManager, meterManagerService)
			if err != nil {
				return fmt.Errorf("failed to create config meters in database: %w", err)
			}
		}

		return nil
	}
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
		return meter.Key
	})

	meters, err := meter.ListAll(ctx, meterService, meter.ListMetersParams{
		Namespace:  namespaceManager.GetDefaultNamespace(),
		SlugFilter: &configMeterSlugs,
	})
	if err != nil {
		return fmt.Errorf("failed to list meters: %w", err)
	}

	metersBySlug := lo.KeyBy(meters, func(meter meter.Meter) string {
		return meter.Key
	})

	for _, configMeter := range configMeters {
		configMeter.Namespace = namespaceManager.GetDefaultNamespace()

		// Backfill the name if it's missing
		if configMeter.Name == "" {
			configMeter.Name = configMeter.Key
		}

		// Create the meter if it doesn't exist
		if dbMeter, ok := metersBySlug[configMeter.Key]; !ok {
			_, err := meterService.CreateMeter(ctx, meter.CreateMeterInput{
				Namespace:     configMeter.Namespace,
				Key:           configMeter.Key,
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

			logger.InfoContext(ctx, "created meter in database", "meter", configMeter.Key)
		} else {
			if err := dbMeter.Equal(*configMeter); err != nil {
				return fmt.Errorf("meter %s in database is not equal to the meter in config: %w", dbMeter.Key, err)
			}

			logger.InfoContext(ctx, "meter in config already exists in database", "meter", configMeter.Key)
		}
	}

	return nil
}
