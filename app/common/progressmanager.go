package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/progressmanager"
	"github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
)

var ProgressManager = wire.NewSet(
	NewProgressManager,
)

// NewProgressManager creates a new progress manager service.
func NewProgressManager(logger *slog.Logger, conf config.ProgressManagerConfiguration) (progressmanager.Service, error) {
	if !conf.Enabled {
		return adapter.NewNoop(), nil
	}

	redisClient, err := conf.Redis.NewClient()
	if err != nil {
		return nil, fmt.Errorf("progress manager redis: %w", err)
	}

	pm, err := adapter.New(adapter.Config{
		Expiration: conf.Expiration,
		KeyPrefix:  conf.KeyPrefix,
		Logger:     logger,
		Redis:      redisClient,
	})
	if err != nil {
		return nil, fmt.Errorf("progress manager adapter: %w", err)
	}

	return pm, nil
}
