package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse/materialized_view"
	"github.com/openmeterio/openmeter/openmeter/streaming/clickhouse/raw_events"
)

var Streaming = wire.NewSet(
	NewStreamingConnector,
)

func NewStreamingConnector(
	ctx context.Context,
	conf config.AggregationConfiguration,
	clickHouse clickhouse.Conn,
	meterRepository meter.Repository,
	logger *slog.Logger,
) (streaming.Connector, error) {
	var (
		connector streaming.Connector
		err       error
	)

	switch conf.Engine {
	case config.AggregationEngineClickHouseRaw:
		connector, err = raw_events.NewConnector(ctx, raw_events.ConnectorConfig{
			ClickHouse:          clickHouse,
			Database:            conf.ClickHouse.Database,
			EventsTableName:     conf.EventsTableName,
			Logger:              logger,
			AsyncInsert:         conf.AsyncInsert,
			AsyncInsertWait:     conf.AsyncInsertWait,
			InsertQuerySettings: conf.InsertQuerySettings,
		})
		if err != nil {
			return nil, fmt.Errorf("init clickhouse raw engine: %w", err)
		}

	case config.AggregationEngineClickHouseMV:
		connector, err = materialized_view.NewConnector(ctx, materialized_view.ConnectorConfig{
			ClickHouse:          clickHouse,
			Database:            conf.ClickHouse.Database,
			EventsTableName:     conf.EventsTableName,
			Logger:              logger,
			AsyncInsert:         conf.AsyncInsert,
			AsyncInsertWait:     conf.AsyncInsertWait,
			InsertQuerySettings: conf.InsertQuerySettings,

			Meters:               meterRepository,
			PopulateMeter:        conf.PopulateMeter,
			CreateOrReplaceMeter: conf.CreateOrReplaceMeter,
			QueryRawEvents:       conf.QueryRawEvents,
		})
		if err != nil {
			return nil, fmt.Errorf("init clickhouse mv engine: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid aggregation engine: %s", conf.Engine)
	}

	return connector, nil
}
