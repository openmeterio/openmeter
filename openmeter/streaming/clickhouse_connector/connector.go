package clickhouse_connector

import (
	"github.com/openmeterio/openmeter/internal/streaming/clickhouse_connector"
)

// ClickhouseConnector implements `ingest.Connector“ and `namespace.Handler interfaces.
type ClickhouseConnector = clickhouse_connector.ClickhouseConnector

type ClickhouseConnectorConfig = clickhouse_connector.ClickhouseConnectorConfig

func NewClickhouseConnector(config ClickhouseConnectorConfig) (*ClickhouseConnector, error) {
	return clickhouse_connector.NewClickhouseConnector(config)
}
