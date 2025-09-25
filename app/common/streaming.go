package common

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/namespace"
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
	namespaceManager *namespace.Manager,
) (streaming.Connector, error) {
	connector, err := clickhouseconnector.New(ctx, clickhouseconnector.Config{
		ClickHouse:          clickHouse,
		Database:            conf.ClickHouse.Database,
		EventsTableName:     conf.EventsTableName,
		Logger:              logger,
		AsyncInsert:         conf.AsyncInsert,
		AsyncInsertWait:     conf.AsyncInsertWait,
		InsertQuerySettings: conf.InsertQuerySettings,
		MeterQuerySettings:  conf.MeterQuerySettings,
		EnablePrewhere:      conf.EnablePrewhere,
		ProgressManager:     progressmanager,
	})
	if err != nil {
		return nil, fmt.Errorf("init clickhouse connector: %w", err)
	}

	err = namespaceManager.RegisterHandler(connector)
	if err != nil {
		return nil, fmt.Errorf("failed to register streaming namespace handler: %w", err)
	}

	return connector, nil
}
