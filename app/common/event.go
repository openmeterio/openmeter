package common

import (
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
)

var Event = wire.NewSet(
	wire.FieldsOf(new(config.Configuration), "Events"),
)
