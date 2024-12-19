package common

import (
	"github.com/google/wire"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var MeterInMemory = wire.NewSet(
	wire.FieldsOf(new(config.Configuration), "Meters"),
	wire.Bind(new(meter.Repository), new(*meter.InMemoryRepository)),

	NewInMemoryRepository,
)

func NewInMemoryRepository(meters []*models.Meter) *meter.InMemoryRepository {
	return meter.NewInMemoryRepository(slicesx.Map(meters, lo.FromPtr[models.Meter]))
}
