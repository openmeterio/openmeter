package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/progressmanager"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	clickhouseconnector "github.com/openmeterio/openmeter/openmeter/streaming/clickhouse"
)

var Streaming = wire.NewSet(
	NewStreamingConnector,
)

func NewStreamingConnector(
	ctx context.Context,
	conf config.AggregationConfiguration,
	clickHouse clickhouse.Conn,
	logger *slog.Logger,
	progressmanager progressmanager.Service,
) (streaming.Connector, error) {
<<<<<<< HEAD
	connector, err := clickhouseconnector.New(ctx, clickhouseconnector.Config{
		ClickHouse:          clickHouse,
		Database:            conf.ClickHouse.Database,
		EventsTableName:     conf.EventsTableName,
		Logger:              logger,
		AsyncInsert:         conf.AsyncInsert,
		AsyncInsertWait:     conf.AsyncInsertWait,
		InsertQuerySettings: conf.InsertQuerySettings,
		ProgressManager:     progressmanager,
	})
	if err != nil {
		return nil, fmt.Errorf("init clickhouse connector: %w", err)
=======
	var (
		connector streaming.Connector
		err       error
	)

	switch conf.Engine {
	case config.AggregationEngineClickHouseRaw:
		connector, err = raw_events.NewConnector(ctx, raw_events.ConnectorConfig{
			ClickHouse:                            clickHouse,
			Database:                              conf.ClickHouse.Database,
			EventsTableName:                       conf.EventsTableName,
			Logger:                                logger,
			AsyncInsert:                           conf.AsyncInsert,
			AsyncInsertWait:                       conf.AsyncInsertWait,
			InsertQuerySettings:                   conf.InsertQuerySettings,
			ProgressManager:                       progressmanager,
			QueryCacheEnabled:                     conf.QueryCache.Enabled,
			QueryCacheMinimumCacheableQueryPeriod: conf.QueryCache.MinimumCacheableQueryPeriod,
			QueryCacheMinimumCacheableUsageAge:    conf.QueryCache.MinimumCacheableUsageAge,
		})
		if err != nil {
			return nil, fmt.Errorf("init clickhouse raw engine: %w", err)
		}

	case config.AggregationEngineClickHouseMV:
		connector, err = materialized_view.NewConnector(ctx, materialized_view.ConnectorConfig{
			ClickHouse:           clickHouse,
			Database:             conf.ClickHouse.Database,
			EventsTableName:      conf.EventsTableName,
			Logger:               logger,
			AsyncInsert:          conf.AsyncInsert,
			AsyncInsertWait:      conf.AsyncInsertWait,
			InsertQuerySettings:  conf.InsertQuerySettings,
			PopulateMeter:        conf.PopulateMeter,
			CreateOrReplaceMeter: conf.CreateOrReplaceMeter,
			QueryRawEvents:       conf.QueryRawEvents,
			ProgressManager:      progressmanager,
		})
		if err != nil {
			return nil, fmt.Errorf("init clickhouse mv engine: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid aggregation engine: %s", conf.Engine)
>>>>>>> 4c356376 (feat(meter): query cache configurable)
	}

	return connector, nil
}
