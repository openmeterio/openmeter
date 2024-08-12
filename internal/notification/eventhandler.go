package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

const (
	DefaultEventHandlerWorkers = 5
)

type EventHandler interface {
	Dispatcher

	Start() error
	Close() error
}

type Reconviler interface {
	Reconcile(ctx context.Context)
}

type Dispatcher interface {
	Dispatch(Event)
}

type EventHandlerConfig struct {
	Repository Repository
	Webhook    webhook.Handler
	Logger     *slog.Logger
}

func (c EventHandlerConfig) Validate() error {
	if c.Repository == nil {
		return fmt.Errorf("repository is required")
	}

	if c.Webhook == nil {
		return fmt.Errorf("webhook is required")
	}

	if c.Logger == nil {
		c.Logger = slog.Default()
	}

	return nil
}

var _ EventHandler = (*handler)(nil)

type handler struct {
	repo    Repository
	webhook webhook.Handler

	logger *slog.Logger

	stopCh chan struct{}
}

func (h *handler) Start() error {
	// FIXME: start reconciler in background
	return nil
}

func (h *handler) Close() error {
	close(h.stopCh)

	return nil
}

func (h *handler) reconcilePending(ctx context.Context, status EventDeliveryStatus) error {
	// FIXME: implement
	return nil
}

func (h *handler) reconcileSending(ctx context.Context, status EventDeliveryStatus) error {
	// FIXME: implement
	return nil
}

func (h *handler) Reconcile(ctx context.Context) error {
	statuses, err := h.repo.ListEventsDeliveryStatus(ctx, ListEventsDeliveryStatusInput{
		Page: pagination.Page{},
		States: []EventDeliveryStatusState{
			EventDeliveryStatusStatePending,
			EventDeliveryStatusStateSending,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to fetch notification delivery statuses for reconciliation: %w", err)
	}

	for _, status := range statuses.Items {
		switch status.State {
		case EventDeliveryStatusStatePending:
			err = h.reconcilePending(ctx, status)
		case EventDeliveryStatusStateSending:
			err = h.reconcileSending(ctx, status)
		}

		if err != nil {
			return fmt.Errorf("failed to reconcile notification delivery status state: %w", err)
		}
	}

	return nil
}

func (h *handler) dispatchWebhook(ctx context.Context, event Event) error {
	channelIDs := slicesx.Map(event.Rule.Channels, func(channel Channel) string {
		return channel.ID
	})

	sendIn := webhook.SendMessageInput{
		Namespace: event.Namespace,
		EventID:   event.ID,
		EventType: string(event.Type),
		Channels:  channelIDs,
	}

	switch event.Type {
	case EventTypeBalanceThreshold:
		payload := event.Payload.AsNotificationEventBalanceThresholdPayload(event.ID, event.CreatedAt)
		payloadMap, err := PayloadToMapInterface(payload)
		if err != nil {
			return fmt.Errorf("failed to cast event payload: %w", err)
		}

		sendIn.Payload = payloadMap
	default:
		return fmt.Errorf("unknown event type: %s", event.Type)
	}

	logger := h.logger.With("eventID", event.ID, "eventType", event.Type)

	stateReason := ""
	state := EventDeliveryStatusStateSending

	_, err := h.webhook.SendMessage(ctx, sendIn)
	if err != nil {
		logger.Error("failed to send webhook message: error returned by webhook service", "error", err)
		stateReason = "failed to send webhook message: error returned by webhook service"
		state = EventDeliveryStatusStateFailed
	}

	for _, channelID := range channelIDs {
		_, err = h.repo.UpdateEventDeliveryStatus(ctx, UpdateEventDeliveryStatusInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: event.Namespace,
			},
			State:     state,
			Reason:    stateReason,
			EventID:   event.ID,
			ChannelID: channelID,
		})
		if err != nil {
			return fmt.Errorf("failed to update event delivery: %w", err)
		}
	}

	return nil
}

func (h *handler) Dispatch(event Event) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logger := h.logger.With("eventID", event.ID, "eventType", event.Type)

		var err error
		for _, channelType := range event.ChannelTypes() {
			switch channelType {
			case ChannelTypeWebhook:
				err = h.dispatchWebhook(ctx, event)
			default:
				h.logger.Error("unknown channel type", "type", channelType)
			}

			if err != nil {
				logger.Error("failed to dispatch event", "error", err)
			}
		}
	}()
}

func NewEventHandler(config EventHandlerConfig) (EventHandler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &handler{
		repo:    config.Repository,
		webhook: config.Webhook,
		stopCh:  make(chan struct{}),
	}, nil
}
