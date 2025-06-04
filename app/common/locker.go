package common

import (
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

var Lockr = wire.NewSet(
	NewLocker,
)

func NewLocker(
	logger *slog.Logger,
) (*lockr.Locker, error) {
	return lockr.NewLocker(&lockr.LockerConfig{
		Logger: logger,
	})
}
