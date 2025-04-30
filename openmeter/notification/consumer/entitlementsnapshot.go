package consumer

import (
	"context"
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
		return b.handleAsSnapshotEvent(ctx, event)
	}

	if b.isEntitlementResetEvent(event) {
		return b.handleAsEntitlementResetEvent(ctx, event)
	}

	return nil
}
