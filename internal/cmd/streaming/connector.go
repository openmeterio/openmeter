package streaming

import (
	"fmt"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/streaming/clickhouse_connector"
)

func GetStreaming(config config.Configuration, clickHouseClient clickhouse.Conn, meterRepository meter.Repository, logger *slog.Logger) (*clickhouse_connector.ClickhouseConnector, error) {
	streamingConnector, err := clickhouse_connector.NewClickhouseConnector(clickhouse_connector.ClickhouseConnectorConfig{
		Logger:               logger,
		ClickHouse:           clickHouseClient,
		Database:             config.Aggregation.ClickHouse.Database,
		Meters:               meterRepository,
		CreateOrReplaceMeter: config.Aggregation.CreateOrReplaceMeter,
		PopulateMeter:        config.Aggregation.PopulateMeter,
	})
	if err != nil {
		return nil, fmt.Errorf("init clickhouse streaming: %w", err)
	}

	return streamingConnector, nil
}
