package consumer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/notification"
)

type EntitlementSnapshotHandler struct {
	Notification notification.Service
	Logger       *slog.Logger
}

func (b *EntitlementSnapshotHandler) Handle(ctx context.Context, event snapshot.SnapshotEvent) error {
	if b.isBalanceThresholdEvent(event) {
		if err := b.handleAsSnapshotEvent(ctx, event); err != nil {
			return fmt.Errorf("failed to handle as snapshot event: %w", err)
		}
	}

	if b.isEntitlementResetEvent(event) {
		if err := b.handleAsEntitlementResetEvent(ctx, event); err != nil {
			return fmt.Errorf("failed to handle as entitlement reset event: %w", err)
		}
	}

	return nil
}
