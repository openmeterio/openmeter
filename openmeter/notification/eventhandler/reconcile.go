package eventhandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/notification"
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

func (h *Handler) Reconcile(ctx context.Context) error {
	events, err := h.repo.ListEvents(ctx, notification.ListEventsInput{
		Page: pagination.Page{},
		DeliveryStatusStates: []notification.EventDeliveryStatusState{
			notification.EventDeliveryStatusStatePending,
			notification.EventDeliveryStatusStateSending,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to fetch notification delivery statuses for reconciliation: %w", err)
	}

	for _, event := range events.Items {
		var errs error
		for _, state := range notification.DeliveryStatusStates(event.DeliveryStatus) {
			switch state {
			case notification.EventDeliveryStatusStatePending:
				if err = h.reconcilePending(ctx, &event); err != nil {
					errs = errors.Join(errs, err)
				}
			case notification.EventDeliveryStatusStateSending:
				if err = h.reconcileSending(ctx, &event); err != nil {
					errs = errors.Join(errs, err)
				}
			case notification.EventDeliveryStatusStateFailed:
				if err = h.reconcileFailed(ctx, &event); err != nil {
					errs = errors.Join(errs, err)
				}
			}
		}

		if errs != nil {
			return fmt.Errorf("failed to reconcile notification event: %w", errs)
		}
	}

	return nil
}
