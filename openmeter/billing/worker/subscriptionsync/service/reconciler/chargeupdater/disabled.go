package chargeupdater

import (
	"context"
	"fmt"
	"log/slog"
)

type disabledUpdater struct {
	logger *slog.Logger
}

func NewDisabled(logger *slog.Logger) Updater {
	return &disabledUpdater{
		logger: logger,
	}
}

func (u *disabledUpdater) ApplyPatches(ctx context.Context, namespace string, patches []Patch) error {
	if len(patches) == 0 {
		return nil
	}

	return fmt.Errorf("charges are disabled")
}

func (u *disabledUpdater) LogPatches(patches []Patch) {
	for _, patch := range patches {
		patch.Log(u.logger)
	}
}
