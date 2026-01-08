package streaming

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

type TableEngine interface {
	IsOperational(meter.Meter) bool
	QueryMeter(ctx context.Context, namespace string, meter meter.Meter, params QueryParams) ([]meter.MeterQueryRow, error)
	Type() string
}

type TableEngineRegistry interface {
	RegisterTableEngine(tableEngine TableEngine)
}
