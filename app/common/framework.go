package common

import "github.com/google/wire"

var Framework = wire.NewSet(
	wire.Struct(new(GlobalInitializer), "*"),
	wire.Struct(new(Runner), "*"),
)
