package common

import (
	"github.com/google/wire"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var MeterInMemory = wire.NewSet(
	wire.FieldsOf(new(config.Configuration), "Meters"),

	NewInMemoryRepository,
)

func NewInMemoryRepository(meters []*models.Meter) meter.Repository {
	return meter.NewInMemoryRepository(slicesx.Map(meters, lo.FromPtr[models.Meter]))
}
