package noop

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
)

var _ webhook.Handler = (*noopHandler)(nil)

// noopHandler is a no-op implementation of the webhook handler.
type noopHandler struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) webhook.Handler {
	return &noopHandler{
		logger: logger.WithGroup("noophandler"),
	}
}

func (h noopHandler) RegisterEventTypes(ctx context.Context, params webhook.RegisterEventTypesInputs) error {
	for _, eventType := range params.EventTypes {
		h.logger.InfoContext(ctx, "registering event types", "event_type", eventType)
	}

	return nil
}

func (h noopHandler) CreateWebhook(ctx context.Context, params webhook.CreateWebhookInput) (*webhook.Webhook, error) {
	h.logger.InfoContext(ctx, "creating webhook", "params", params)

	return nil, fmt.Errorf("create webhook: not implemented")
}

func (h noopHandler) UpdateWebhook(ctx context.Context, params webhook.UpdateWebhookInput) (*webhook.Webhook, error) {
	h.logger.InfoContext(ctx, "updating webhook", "params", params)

	return nil, fmt.Errorf("update webhook: not implemented")
}

func (h noopHandler) UpdateWebhookChannels(ctx context.Context, params webhook.UpdateWebhookChannelsInput) (*webhook.Webhook, error) {
	h.logger.InfoContext(ctx, "updating webhook channels", "params", params)

	return nil, fmt.Errorf("update webhook channels: not implemented")
}

func (h noopHandler) DeleteWebhook(ctx context.Context, params webhook.DeleteWebhookInput) error {
	h.logger.InfoContext(ctx, "deleting webhook", "params", params)

	return nil
}

func (h noopHandler) GetWebhook(ctx context.Context, params webhook.GetWebhookInput) (*webhook.Webhook, error) {
	h.logger.InfoContext(ctx, "getting webhook", "params", params)

	return nil, fmt.Errorf("get webhook: not implemented")
}

func (h noopHandler) ListWebhooks(ctx context.Context, params webhook.ListWebhooksInput) ([]webhook.Webhook, error) {
	h.logger.InfoContext(ctx, "listing webhooks", "params", params)

	return []webhook.Webhook{}, nil
}

func (h noopHandler) SendMessage(ctx context.Context, params webhook.SendMessageInput) (*webhook.Message, error) {
	h.logger.InfoContext(ctx, "sending message", "params", params)

	return nil, fmt.Errorf("send message: not implemented")
}
