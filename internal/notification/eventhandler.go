package notification

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/internal/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

const DefaultReconcileInterval = 15 * time.Second

type EventHandler interface {
	EventDispatcher
	EventReconciler

	Start() error
	Close() error
}

type EventReconciler interface {
	Reconcile(ctx context.Context) error
}

type EventDispatcher interface {
	Dispatch(*Event) error
}

type EventHandlerConfig struct {
	Repository        Repository
	Webhook           webhook.Handler
	Logger            *slog.Logger
	ReconcileInterval time.Duration
}

func (c *EventHandlerConfig) Validate() error {
	if c.Repository == nil {
		return fmt.Errorf("repository is required")
	}

	if c.Webhook == nil {
		return fmt.Errorf("webhook is required")
	}

	return nil
}

var _ EventHandler = (*handler)(nil)

type handler struct {
	repo    Repository
	webhook webhook.Handler
	logger  *slog.Logger

	reconcileInterval time.Duration

	stopCh chan struct{}
}

func (h *handler) Start() error {
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ticker := time.NewTicker(h.reconcileInterval)
		defer ticker.Stop()

		logger := h.logger.WithGroup("reconciler")

		for {
			select {
			case <-h.stopCh:
				logger.Debug("close event received: stopping reconciler")
				return
			case <-ticker.C:
				if err := h.Reconcile(ctx); err != nil {
					logger.Error("failed to reconcile event(s)", "error", err)
				}
			}
		}
	}()

	return nil
}

func (h *handler) Close() error {
	close(h.stopCh)

	return nil
}

func (h *handler) reconcilePending(ctx context.Context, event *Event) error {
	return h.dispatch(ctx, event)
}

func (h *handler) reconcileSending(_ context.Context, _ *Event) error {
	// NOTE(chrisgacsal): implement when EventDeliveryStatusStateSending state is need to be handled
	return nil
}

func (h *handler) reconcileFailed(_ context.Context, _ *Event) error {
	// NOTE(chrisgacsal): reconcile failed events when adding support for retry on event delivery failure
	return nil
}

func (h *handler) Reconcile(ctx context.Context) error {
	events, err := h.repo.ListEvents(ctx, ListEventsInput{
		Page: pagination.Page{},
		DeliveryStatusStates: []EventDeliveryStatusState{
			EventDeliveryStatusStatePending,
			EventDeliveryStatusStateSending,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to fetch notification delivery statuses for reconciliation: %w", err)
	}

	for _, event := range events.Items {
		var errs error
		for _, state := range event.DeliveryStates() {
			switch state {
			case EventDeliveryStatusStatePending:
				if err = h.reconcilePending(ctx, &event); err != nil {
					errs = errors.Join(errs, err)
				}
			case EventDeliveryStatusStateSending:
				if err = h.reconcileSending(ctx, &event); err != nil {
					errs = errors.Join(errs, err)
				}
			case EventDeliveryStatusStateFailed:
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

func (h *handler) dispatchWebhook(ctx context.Context, event *Event) error {
	channelIDs := slicesx.Map(event.Rule.Channels, func(channel Channel) string {
		return channel.ID
	})

	sendIn := webhook.SendMessageInput{
		Namespace: event.Namespace,
		EventID:   event.ID,
		EventType: string(event.Type),
		Channels:  []string{event.Rule.ID},
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

	var stateReason string
	state := EventDeliveryStatusStateSuccess
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

func (h *handler) dispatch(ctx context.Context, event *Event) error {
	var errs error

	for _, channelType := range event.ChannelTypes() {
		var err error

		switch channelType {
		case ChannelTypeWebhook:
			err = h.dispatchWebhook(ctx, event)
		default:
			err = fmt.Errorf("unknown channel type: %s", channelType)
		}

		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (h *handler) Dispatch(event *Event) error {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := h.dispatch(ctx, event); err != nil {
			h.logger.Warn("failed to dispatch event", "eventID", event.ID, "error", err)
		}
	}()

	return nil
}

func NewEventHandler(config EventHandlerConfig) (EventHandler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if config.ReconcileInterval == 0 {
		config.ReconcileInterval = DefaultReconcileInterval
	}

	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	return &handler{
		repo:              config.Repository,
		webhook:           config.Webhook,
		reconcileInterval: config.ReconcileInterval,
		logger:            config.Logger,
		stopCh:            make(chan struct{}),
	}, nil
}
