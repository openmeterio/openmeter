package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s Service) UpdateEventDeliveryStatus(ctx context.Context, params notification.UpdateEventDeliveryStatusInput) (*notification.EventDeliveryStatus, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	fn := func(ctx context.Context) (*notification.EventDeliveryStatus, error) {
		return s.adapter.UpdateEventDeliveryStatus(ctx, params)
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s Service) ListEventsDeliveryStatus(ctx context.Context, params notification.ListEventsDeliveryStatusInput) (notification.ListEventsDeliveryStatusResult, error) {
	if err := params.Validate(); err != nil {
		return notification.ListEventsDeliveryStatusResult{}, fmt.Errorf("invalid params: %w", err)
	}

	return s.adapter.ListEventsDeliveryStatus(ctx, params)
}

func (s Service) GetEventDeliveryStatus(ctx context.Context, params notification.GetEventDeliveryStatusInput) (*notification.EventDeliveryStatus, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	return s.adapter.GetEventDeliveryStatus(ctx, params)
}
