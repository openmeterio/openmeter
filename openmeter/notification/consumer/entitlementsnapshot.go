package consumer

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
)

type EntitlementSnapshotHandler struct {
	Notification notification.Service
	Logger       *slog.Logger
	Tracer       trace.Tracer
}

func (b *EntitlementSnapshotHandler) Handle(ctx context.Context, event snapshot.SnapshotEvent) error {
	return tracex.StartWithNoValue(ctx, b.Tracer, "notification.consumer.entitlement_snapshot",
		trace.WithAttributes(
			attribute.String("namespace", event.Namespace.ID),
			attribute.String("entitlement.id", event.Entitlement.ID),
			attribute.String("feature.key", event.Entitlement.FeatureKey),
			attribute.String("feature.id", event.Entitlement.FeatureID),
			attribute.String("operation", string(event.Operation)),
		),
	).Wrap(func(ctx context.Context) error {
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
	})
}
