package common

import (
	"github.com/google/wire"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meter/adapter"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var MeterInMemory = wire.NewSet(
	wire.FieldsOf(new(config.Configuration), "Meters"),

	NewMeterService,
)

func NewMeterService(
	meters []*meter.Meter,
) (meter.Service, error) {
	staticMeters := slicesx.Map(meters, lo.FromPtr[meter.Meter])
	return adapter.New(staticMeters)
}
