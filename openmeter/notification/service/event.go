package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s Service) ListEvents(ctx context.Context, params notification.ListEventsInput) (notification.ListEventsResult, error) {
	if err := params.Validate(); err != nil {
		return notification.ListEventsResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return s.adapter.ListEvents(ctx, params)
}

func (s Service) GetEvent(ctx context.Context, params notification.GetEventInput) (*notification.Event, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return s.adapter.GetEvent(ctx, params)
}

func (s Service) CreateEvent(ctx context.Context, params notification.CreateEventInput) (*notification.Event, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	fn := func(ctx context.Context) (*notification.Event, error) {
		logger := s.logger.WithGroup("event").With(
			"operation", "create",
			"namespace", params.Namespace,
		)

		logger.Debug("creating event")

		rule, err := s.adapter.GetRule(ctx, notification.GetRuleInput{
			Namespace: params.Namespace,
			ID:        params.RuleID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get rule: %w", err)
		}

		if rule.DeletedAt != nil {
			return nil, notification.NotFoundError{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
					ID:        params.RuleID,
				},
			}
		}

		if rule.Disabled {
			return nil, models.NewGenericValidationError(errors.New("failed to send event: rule is disabled"))
		}

		event, err := s.adapter.CreateEvent(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create event: %w", err)
		}

		if err = s.eventHandler.Dispatch(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to dispatch event: %w", err)
		}

		return event, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s Service) ResendEvent(ctx context.Context, params notification.ResendEventInput) error {
	if err := params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	fn := func(ctx context.Context) error {
		event, err := s.adapter.GetEvent(ctx, notification.GetEventInput{
			Namespace: params.Namespace,
			ID:        params.ID,
		})
		if err != nil {
			return fmt.Errorf("failed to get event: %w", err)
		}

		var errs []error

		channelsByID := lo.SliceToMap(event.Rule.Channels, func(item notification.Channel) (string, notification.Channel) {
			return item.ID, item
		})

		for _, channelID := range params.Channels {
			channel, ok := channelsByID[channelID]
			if !ok {
				errs = append(errs, fmt.Errorf("channel %s not found", channelID))
				continue
			}

			if channel.Disabled {
				errs = append(errs, fmt.Errorf("channel %s is disabled", channelID))
			}
		}

		if len(errs) > 0 {
			return models.NewGenericValidationError(errors.Join(errs...))
		}

		allowedStates := []notification.EventDeliveryStatusState{
			notification.EventDeliveryStatusStateSending,
			notification.EventDeliveryStatusStateSuccess,
			notification.EventDeliveryStatusStateFailed,
		}

		now := clock.Now()

		for _, status := range event.DeliveryStatus {
			if !lo.Contains(allowedStates, status.State) {
				continue
			}

			// If there are params.Channels, only resend to those channels.
			if len(params.Channels) > 0 && !lo.Contains(params.Channels, status.ChannelID) {
				continue
			}

			// Don't resend to disabled channels.
			channel, ok := channelsByID[status.ChannelID]
			if ok && channel.Disabled {
				continue
			}

			annotations := lo.Assign(status.Annotations, models.Annotations{
				notification.AnnotationEventResendTimestamp: now.UTC().Format(time.RFC3339),
			})

			_, err = s.adapter.UpdateEventDeliveryStatus(ctx, notification.UpdateEventDeliveryStatusInput{
				NamespacedID: status.NamespacedID,
				State:        notification.EventDeliveryStatusStateResending,
				Reason:       "event re-send was triggered",
				Annotations:  annotations,
				NextAttempt:  nil,
				Attempts:     status.Attempts,
			})
			if err != nil {
				return fmt.Errorf("failed to resend event: %w", err)
			}
		}

		return nil
	}

	return transaction.RunWithNoValue(ctx, s.adapter, fn)
}
