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
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/progressmanager"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	clickhouseconnector "github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
	streamingretry "github.com/openmeterio/openmeter/openmeter/streaming/retry"
)

var Streaming = wire.NewSet(
	NewClickHouseStreamingConnector,
	NewStreamingConnector,
)

// NewClickHouseStreamingConnector provides the concrete ClickHouse connector separately
// from the streaming.Connector interface: the meter cache lifecycle reconciler drives the
// connector's cache manager surface (EnsureMeterCache, DropMeterCache, ListActualViews),
// which the interface — and the retry wrapper NewStreamingConnector may add — does not
// carry.
func NewClickHouseStreamingConnector(
	ctx context.Context,
	conf config.AggregationConfiguration,
	clickHouse clickhouse.Conn,
	logger *slog.Logger,
	progressmanager progressmanager.Service,
	meter metric.Meter,
	tracer trace.Tracer,
) (*clickhouseconnector.Connector, error) {
	connector, err := clickhouseconnector.New(ctx, clickhouseconnector.Config{
		ClickHouse:             clickHouse,
		Database:               conf.ClickHouse.Database,
		EventsTableName:        conf.EventsTableName,
		Logger:                 logger,
		AsyncInsert:            conf.AsyncInsert,
		AsyncInsertWait:        conf.AsyncInsertWait,
		InsertQuerySettings:    conf.InsertQuerySettings,
		MeterQuerySettings:     conf.MeterQuerySettings,
		EnablePrewhere:         conf.EnablePrewhere,
		EnableDecimalPrecision: conf.EnableDecimalPrecision,
		ProgressManager:        progressmanager,
		Cache:                  mapAggregationCacheConfig(conf.Cache),
		Meter:                  meter,
		Tracer:                 tracer,
	})
	if err != nil {
		return nil, fmt.Errorf("init clickhouse connector: %w", err)
	}

	return connector, nil
}

func NewStreamingConnector(
	conf config.AggregationConfiguration,
	clickHouseConnector *clickhouseconnector.Connector,
	logger *slog.Logger,
	namespaceManager *namespace.Manager,
) (streaming.Connector, error) {
	var connector streaming.Connector = clickHouseConnector
	var err error

	if conf.ClickHouse.Retry.Enabled {
		connector, err = streamingretry.New(streamingretry.Config{
			DownstreamConnector: connector,
			Logger:              logger,
			RetryWaitDuration:   conf.ClickHouse.Retry.RetryWaitDuration,
			MaxTries:            conf.ClickHouse.Retry.MaxTries,
			MaxDelay:            conf.ClickHouse.Retry.MaxDelay,
		})
		if err != nil {
			return nil, fmt.Errorf("init retry connector: %w", err)
		}
	}

	err = namespaceManager.RegisterHandler(connector)
	if err != nil {
		return nil, fmt.Errorf("failed to register streaming namespace handler: %w", err)
	}

	return connector, nil
}

// mapAggregationCacheConfig maps the validated app config into the clickhouse connector's
// own CacheConfig type. The connector package must not import app/config (app/config
// already depends on lower-level domain packages, so the reverse import would create a
// cycle across the DI boundary), so this mapping is the only place the two types meet.
func mapAggregationCacheConfig(conf config.AggregationCacheConfiguration) clickhouseconnector.CacheConfig {
	return clickhouseconnector.CacheConfig{
		Enabled:             conf.Enabled,
		RefreshInterval:     conf.RefreshInterval,
		MinimumUsageAge:     conf.MinimumUsageAge,
		WindowSize:          clickhouseconnector.CacheGrain(conf.WindowSize),
		MeterQueryThreshold: conf.MeterQueryThreshold,
	}
}
