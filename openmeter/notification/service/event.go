package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/notification"
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
