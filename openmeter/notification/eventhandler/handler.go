package eventhandler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Config struct {
	Repository        notification.Repository
	Webhook           webhook.Handler
	Logger            *slog.Logger
	ReconcileInterval time.Duration
}

func (c *Config) Validate() error {
	if c.Repository == nil {
		return fmt.Errorf("repository is required")
	}

	if c.Webhook == nil {
		return fmt.Errorf("webhook is required")
	}

	return nil
}

var _ notification.EventHandler = (*Handler)(nil)

type Handler struct {
	repo    notification.Repository
	webhook webhook.Handler
	logger  *slog.Logger

	reconcileInterval time.Duration

	stopCh chan struct{}
}

func (h *Handler) Start() error {
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
					logger.ErrorContext(ctx, "failed to reconcile event(s)", "error", err)
				}
			}
		}
	}()

	return nil
}

func (h *Handler) Close() error {
	close(h.stopCh)

	return nil
}

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

func (h *Handler) dispatchWebhook(ctx context.Context, event *notification.Event) error {
	sendIn := webhook.SendMessageInput{
		Namespace: event.Namespace,
		EventID:   event.ID,
		EventType: string(event.Type),
		Channels:  []string{event.Rule.ID},
	}

	switch event.Type {
	case notification.EventTypeBalanceThreshold:
		payload := event.Payload.AsNotificationEventBalanceThresholdPayload(event.ID, event.CreatedAt)
		payloadMap, err := notification.PayloadToMapInterface(payload)
		if err != nil {
			return fmt.Errorf("failed to cast event payload: %w", err)
		}

		sendIn.Payload = payloadMap
	default:
		return fmt.Errorf("unknown event type: %s", event.Type)
	}

	logger := h.logger.With("eventID", event.ID, "eventType", event.Type)

	var stateReason string
	state := notification.EventDeliveryStatusStateSuccess
	_, err := h.webhook.SendMessage(ctx, sendIn)
	if err != nil {
		logger.ErrorContext(ctx, "failed to send webhook message: error returned by webhook service", "error", err)
		stateReason = "failed to send webhook message: error returned by webhook service"
		state = notification.EventDeliveryStatusStateFailed
	}

	for _, channelID := range notification.ChannelIDsByType(event.Rule.Channels, notification.ChannelTypeWebhook) {
		_, err = h.repo.UpdateEventDeliveryStatus(ctx, notification.UpdateEventDeliveryStatusInput{
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

func (h *Handler) dispatch(ctx context.Context, event *notification.Event) error {
	var errs error

	for _, channelType := range notification.ChannelTypes(event.Rule.Channels) {
		var err error

		switch channelType {
		case notification.ChannelTypeWebhook:
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

func (h *Handler) Dispatch(event *notification.Event) error {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), notification.DefaultDispatchTimeout)
		defer cancel()

		if err := h.dispatch(ctx, event); err != nil {
			h.logger.Warn("failed to dispatch event", "eventID", event.ID, "error", err)
		}
	}()

	return nil
}

func New(config Config) (*Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	if config.ReconcileInterval == 0 {
		config.ReconcileInterval = notification.DefaultReconcileInterval
	}

	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	return &Handler{
		repo:              config.Repository,
		webhook:           config.Webhook,
		reconcileInterval: config.ReconcileInterval,
		logger:            config.Logger,
		stopCh:            make(chan struct{}),
	}, nil
}
