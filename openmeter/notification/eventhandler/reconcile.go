package eventhandler

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (h *Handler) reconcilePending(ctx context.Context, event *notification.Event) error {
	return h.dispatch(ctx, event)
}

func (h *Handler) reconcileSending(_ context.Context, _ *notification.Event) error {
	// NOTE(chrisgacsal): implement when EventDeliveryStatusStateSending state is need to be handled
	return nil
}

func (h *Handler) reconcileFailed(_ context.Context, _ *notification.Event) error {
	// NOTE(chrisgacsal): reconcile failed events when adding support for retry on event delivery failure
	return nil
}

func (h *Handler) reconcileEvent(ctx context.Context, event *notification.Event) error {
	fn := func(ctx context.Context) error {
		var errs []error

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("notification.event.id", event.ID),
			attribute.String("notification.event.namespace", event.Namespace),
		}

		span.SetAttributes(spanAttrs...)

		for _, status := range event.DeliveryStatus {
			span.AddEvent("reconciling event", trace.WithAttributes(spanAttrs...),
				trace.WithAttributes(
					attribute.Stringer("notification.event.delivery_status.state", status.State),
					attribute.String("notification.event.channel.id", status.ChannelID),
				),
			)

			switch status.State {
			case notification.EventDeliveryStatusStatePending:
				if err := h.reconcilePending(ctx, event); err != nil {
					errs = append(errs, err)
				}
			case notification.EventDeliveryStatusStateSending:
				if err := h.reconcileSending(ctx, event); err != nil {
					errs = append(errs, err)
				}
			case notification.EventDeliveryStatusStateFailed:
				if err := h.reconcileFailed(ctx, event); err != nil {
					errs = append(errs, err)
				}
			}
		}

		return errors.Join(errs...)
	}

	return tracex.StartWithNoValue(ctx, h.tracer, "event_handler.reconcile_event").Wrap(fn)
}

const reconcileLockKey = "notification.event_handler.reconcile_lock"

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

			var errs []error

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
					},
				})
				if err != nil {
					return fmt.Errorf("failed to fetch notification delivery statuses for reconciliation: %w", err)
				}

				span.AddEvent("reconciling events", trace.WithAttributes(
					attribute.Int("event_handler.reconcile.count", len(out.Items)),
				))

				for _, deliveryStatus := range out.Items {
					if err = h.reconcileEvent(ctx, &deliveryStatus); err != nil {
						errs = append(errs,
							fmt.Errorf("failed to reconcile notification event [namespace=%s event.id=%s]: %w",
								deliveryStatus.Namespace, deliveryStatus.ID, err),
						)
					}
				}

				if out.TotalCount <= page.PageSize*page.PageNumber || len(out.Items) == 0 {
					break
				}

				page.PageNumber++
			}

			return errors.Join(errs...)
		})
	}

	return tracex.StartWithNoValue(ctx, h.tracer, "event_handler.reconcile").Wrap(fn)
}
