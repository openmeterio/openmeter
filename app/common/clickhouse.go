package common

import (
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/openmeterio/openmeter/app/config"
)

// TODO: add closer function?
func NewClickHouse(conf config.ClickHouseAggregationConfiguration) (clickhouse.Conn, error) {
	conn, err := clickhouse.Open(conf.GetClientOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize clickhouse client: %w", err)
	}

	return conn, nil
}
