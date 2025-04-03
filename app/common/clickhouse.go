package common

import (
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/pkg/framework/clickhouseotel"
)

var ClickHouse = wire.NewSet(
	NewClickHouse,
)

// TODO: add closer function?
func NewClickHouse(conf config.ClickHouseAggregationConfiguration, tracer trace.Tracer) (clickhouse.Conn, error) {
	conn, err := clickhouse.Open(conf.GetClientOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize clickhouse client: %w", err)
	}

	if conf.Tracing {
		conn, err = clickhouseotel.NewClickHouseTracer(clickhouseotel.ClickHouseTracerConfig{
			Conn:   conn,
			Tracer: tracer,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize clickhouse tracer: %w", err)
		}
	}

	return conn, nil
}
