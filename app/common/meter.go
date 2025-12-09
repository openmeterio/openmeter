package common

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/google/wire"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/app/config"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meter/adapter"
	"github.com/openmeterio/openmeter/openmeter/meter/service"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type MeterConfigInitializer = func(ctx context.Context) error

var Meter = wire.NewSet(
	NewMeterService,
	NewMeterAdapter,
)

var MeterManage = wire.NewSet(
	Meter,
	NewMeterManageService,
	NewReservedEventTypePatterns,
)

var MeterManageWithConfigMeters = wire.NewSet(
	wire.FieldsOf(new(config.Configuration), "Meters"),

	Meter,
	NewMeterManageService,
	NewReservedEventTypePatterns,
	NewMeterConfigInitializer,
)

func NewMeterService(
	meterAdapter *adapter.Adapter,
) meter.Service {
	return service.New(meterAdapter)
}

func NewReservedEventTypePatterns(reserved []config.ReservedEventTypePattern) ([]*meter.EventTypePattern, error) {
	var errs []error

	patterns := make([]*meter.EventTypePattern, 0, len(reserved))

	for _, r := range reserved {
		pattern, err := regexp.Compile(r)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid reserved event type pattern %q: %w", r, err))

			continue
		}

		patterns = append(patterns, pattern)
	}

	return patterns, errors.Join(errs...)
}

func NewMeterManageService(
	meterAdapter *adapter.Adapter,
	namespaceManager *namespace.Manager,
	publisher eventbus.Publisher,
	reservedEventTypes []*meter.EventTypePattern,
) meter.ManageService {
	return service.NewManage(
		meterAdapter,
		publisher,
		namespaceManager,
		reservedEventTypes,
	)
}

func NewMeterAdapter(
	logger *slog.Logger,
	db *entdb.Client,
) (*adapter.Adapter, error) {
	return adapter.New(adapter.Config{
		Client: db,
		Logger: logger,
	})
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

	meterList, err := meterService.ListMeters(ctx, meter.ListMetersParams{
		Namespace:  namespaceManager.GetDefaultNamespace(),
		SlugFilter: &configMeterSlugs,
	})
	if err != nil {
		return fmt.Errorf("failed to list meters: %w", err)
	}

	metersBySlug := lo.KeyBy(meterList.Items, func(meter meter.Meter) string {
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
