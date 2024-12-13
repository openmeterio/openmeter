package common

import (
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
)

var Svix = wire.NewSet(
	wire.FieldsOf(new(config.Configuration), "Svix"),
)
