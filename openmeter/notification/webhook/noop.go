package webhook

import (
	"context"
	"fmt"
	"log/slog"
)

var _ Handler = (*noopWebhookHandler)(nil)

// noopWebhookHandler is a no-op implementation of the webhook handler.
type noopWebhookHandler struct {
	logger *slog.Logger
}

func newNoopWebhookHandler(logger *slog.Logger) Handler {
	return &noopWebhookHandler{
		logger: logger.WithGroup("noophandler"),
	}
}

func (h noopWebhookHandler) RegisterEventTypes(ctx context.Context, params RegisterEventTypesInputs) error {
	for _, eventType := range params.EventTypes {
		h.logger.InfoContext(ctx, "registering event types", "event_type", eventType)
	}

	return nil
}

func (h noopWebhookHandler) CreateWebhook(ctx context.Context, params CreateWebhookInput) (*Webhook, error) {
	h.logger.InfoContext(ctx, "creating webhook", "params", params)

	return nil, fmt.Errorf("create webhook: not implemented")
}

func (h noopWebhookHandler) UpdateWebhook(ctx context.Context, params UpdateWebhookInput) (*Webhook, error) {
	h.logger.InfoContext(ctx, "updating webhook", "params", params)

	return nil, fmt.Errorf("update webhook: not implemented")
}

func (h noopWebhookHandler) UpdateWebhookChannels(ctx context.Context, params UpdateWebhookChannelsInput) (*Webhook, error) {
	h.logger.InfoContext(ctx, "updating webhook channels", "params", params)

	return nil, fmt.Errorf("update webhook channels: not implemented")
}

func (h noopWebhookHandler) DeleteWebhook(ctx context.Context, params DeleteWebhookInput) error {
	h.logger.InfoContext(ctx, "deleting webhook", "params", params)

	return nil
}

func (h noopWebhookHandler) GetWebhook(ctx context.Context, params GetWebhookInput) (*Webhook, error) {
	h.logger.InfoContext(ctx, "getting webhook", "params", params)

	return nil, fmt.Errorf("get webhook: not implemented")
}

func (h noopWebhookHandler) ListWebhooks(ctx context.Context, params ListWebhooksInput) ([]Webhook, error) {
	h.logger.InfoContext(ctx, "listing webhooks", "params", params)

	return []Webhook{}, nil
}

func (h noopWebhookHandler) SendMessage(ctx context.Context, params SendMessageInput) (*Message, error) {
	h.logger.InfoContext(ctx, "sending message", "params", params)

	return nil, fmt.Errorf("send message: not implemented")
}
