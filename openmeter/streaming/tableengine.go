package streaming

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

type QueryMeterSQL interface {
	ToCountRowSQL() (string, []interface{})
	ToSQL() (string, []interface{}, error)
	ScanRows(rows driver.Rows) ([]meter.MeterQueryRow, error)
}

type TableEngine interface {
	IsOperational(meter.Meter) bool
	QueryMeter(ctx context.Context, namespace string, meter meter.Meter, params QueryParams) (QueryMeterSQL, error)
	Type() string
}

type TableEngineRegistry interface {
	RegisterTableEngine(tableEngine TableEngine)
}
