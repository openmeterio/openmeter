package common

import (
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
)

var ClickHouse = wire.NewSet(
	NewClickHouse,
)

// TODO: add closer function?
func NewClickHouse(conf config.ClickHouseAggregationConfiguration) (clickhouse.Conn, error) {
	conn, err := clickhouse.Open(conf.GetClientOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize clickhouse client: %w", err)
	}

	return conn, nil
}
