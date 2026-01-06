package eventhandler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (h *Handler) reconcileEvent(ctx context.Context, event *notification.Event) error {
	fn := func(ctx context.Context) error {
		if event == nil {
			return fmt.Errorf("event must not be nil")
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("notification.event.id", event.ID),
			attribute.String("notification.event.namespace", event.Namespace),
		}

		span.SetAttributes(spanAttrs...)

		channelTypes := lo.UniqMap(event.Rule.Channels, func(item notification.Channel, _ int) notification.ChannelType {
			return item.Type
		})

		var errs []error

		for _, channelType := range channelTypes {
			switch channelType {
			case notification.ChannelTypeWebhook:
				if err := h.reconcileWebhookEvent(ctx, event); err != nil {
					errs = append(errs, err)
				}
			default:
				h.logger.ErrorContext(ctx, "unsupported channel type", "type", channelType)
			}
		}

		return errors.Join(errs...)
	}

	return tracex.StartWithNoValue(ctx, h.tracer, "event_handler.reconcile_event").Wrap(fn)
}

const (
	reconcileLockKey = "notification.event_handler.reconcile_lock"

	// nextAttemptDelay is a jitter to delay reconciliation of events to give time for downstream service providers to
	// update their states with the result of the latest attempts which usually happen asynchronously.
	// This way we can limit the number of missing state updates which could happen if we try to reconcile/synchronize
	// states right around the *nextAttempt* time provided the downstream service in the previous reconciliation attempt.
	nextAttemptDelay = 10 * time.Second
)

func (h *Handler) Reconcile(ctx context.Context) error {
	fn := func(ctx context.Context) error {
		return transaction.RunWithNoValue(ctx, h.repo, func(ctx context.Context) error {
			span := trace.SpanFromContext(ctx)

			span.AddEvent("acquiring lock")

			if err := h.lockr.LockForTXWithScopes(ctx, reconcileLockKey); err != nil {
				if errors.Is(err, lockr.ErrLockTimeout) {
					h.logger.WarnContext(ctx, "reconciliation lock is not available, skipping reconciliation")

					return nil
				}

				return fmt.Errorf("failed to acquire reconciliation lock: %w", err)
			}

			span.AddEvent("lock acquired")

			page := pagination.Page{
				PageSize:   50,
				PageNumber: 1,
			}

			for {
				out, err := h.repo.ListEvents(ctx, notification.ListEventsInput{
					Page: page,
					DeliveryStatusStates: []notification.EventDeliveryStatusState{
						notification.EventDeliveryStatusStatePending,
						notification.EventDeliveryStatusStateSending,
						notification.EventDeliveryStatusStateResending,
					},
					NextAttemptBefore: clock.Now().Add(-1 * nextAttemptDelay),
				})
				if err != nil {
					return fmt.Errorf("failed to fetch notification delivery statuses for reconciliation: %w", err)
				}

				span.AddEvent("reconciling events", trace.WithAttributes(
					attribute.Int("event_handler.reconcile.count", len(out.Items)),
				))

				for _, event := range out.Items {
					// TODO: run reconciliation in parallel (goroutines)
					if err = h.reconcileEvent(ctx, &event); err != nil {
						h.logger.ErrorContext(ctx, "failed to reconcile notification event",
							"namespace", event.Namespace,
							"notification.event.id", event.ID,
							"error", err.Error(),
						)
					}
				}

				if out.TotalCount <= page.PageSize*page.PageNumber || len(out.Items) == 0 {
					break
				}

				page.PageNumber++
			}

			return nil
		})
	}

	return tracex.StartWithNoValue(ctx, h.tracer, "event_handler.reconcile").Wrap(fn)
}
