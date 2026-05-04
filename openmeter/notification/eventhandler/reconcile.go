package eventhandler

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/semaphore"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
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

// nextAttemptDelay is a jitter to delay reconciliation of events to give time for downstream service providers to
// update their states with the result of the latest attempts which usually happen asynchronously.
// This way we can limit the number of missing state updates which could happen if we try to reconcile/synchronize
// states right around the *nextAttempt* time provided the downstream service in the previous reconciliation attempt.
const nextAttemptDelay = 10 * time.Second

func (h *Handler) Reconcile(ctx context.Context) error {
	fn := func(ctx context.Context) error {
		span := trace.SpanFromContext(ctx)

		span.AddEvent("acquiring lock")

		span.AddEvent("lock acquired")

		workerPool := semaphore.NewWeighted(h.workerPoolSize)

		wg := sync.WaitGroup{}
		defer func() {
			// Wait for all workers to finish
			wg.Wait()

			h.logger.DebugContext(ctx, "all workers finished")
		}()

		page := pagination.Page{
			PageSize:   50,
			PageNumber: 1,
		}

		nextAttemptBefore := clock.Now().Add(-1 * nextAttemptDelay)

		for {
			out, err := h.repo.ListEvents(ctx, notification.ListEventsInput{
				Page: page,
				DeliveryStatusStates: []notification.EventDeliveryStatusState{
					notification.EventDeliveryStatusStatePending,
					notification.EventDeliveryStatusStateSending,
					notification.EventDeliveryStatusStateResending,
				},
				NextAttemptBefore: nextAttemptBefore,
			})
			if err != nil {
				return fmt.Errorf("failed to fetch notification delivery statuses for reconciliation: %w", err)
			}

			span.AddEvent("reconciling events", trace.WithAttributes(
				attribute.Int("event_handler.reconcile.count", len(out.Items)),
			))

			for _, event := range out.Items {
				err = workerPool.Acquire(ctx, 1)
				if err != nil {
					return fmt.Errorf("failed to acquire worker from pool: %w", err)
				}

				wg.Go(func() {
					defer workerPool.Release(1)

					defer func() {
						if err := recover(); err != nil {
							h.logger.ErrorContext(ctx, "notification event handler worker panicked",
								"error", err,
								"code.stacktrace", string(debug.Stack()))
						}
					}()

					if rErr := h.reconcileEvent(ctx, &event); rErr != nil {
						h.logger.ErrorContext(ctx, "failed to reconcile notification event",
							"namespace", event.Namespace,
							"notification.event.id", event.ID,
							"error", rErr.Error(),
						)
					}
				})
			}

			if out.TotalCount <= page.PageSize*page.PageNumber || len(out.Items) == 0 {
				break
			}

			page.PageNumber++
		}

		return nil
	}

	return tracex.StartWithNoValue(ctx, h.tracer, "event_handler.reconcile").Wrap(fn)
}
