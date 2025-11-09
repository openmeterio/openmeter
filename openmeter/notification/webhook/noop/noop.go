package noop

import (
	"context"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
)

var _ webhook.Handler = (*Handler)(nil)

// Handler is a no-op implementation of the webhook handler.
type Handler struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Handler {
	return &Handler{
		logger: logger.With(slog.String("webhook_handler", "noop")),
	}
}

func (h Handler) RegisterEventTypes(ctx context.Context, params webhook.RegisterEventTypesInputs) error {
	for _, eventType := range params.EventTypes {
		h.logger.InfoContext(ctx, "registering event types", "event_type", eventType)
	}

	return webhook.ErrNotImplemented
}

func (h Handler) CreateWebhook(ctx context.Context, params webhook.CreateWebhookInput) (*webhook.Webhook, error) {
	h.logger.InfoContext(ctx, "creating webhook", "params", params)

	return nil, webhook.ErrNotImplemented
}

func (h Handler) UpdateWebhook(ctx context.Context, params webhook.UpdateWebhookInput) (*webhook.Webhook, error) {
	h.logger.InfoContext(ctx, "updating webhook", "params", params)

	return nil, webhook.ErrNotImplemented
}

func (h Handler) UpdateWebhookChannels(ctx context.Context, params webhook.UpdateWebhookChannelsInput) (*webhook.Webhook, error) {
	h.logger.InfoContext(ctx, "updating webhook channels", "params", params)

	return nil, webhook.ErrNotImplemented
}

func (h Handler) DeleteWebhook(ctx context.Context, params webhook.DeleteWebhookInput) error {
	h.logger.InfoContext(ctx, "deleting webhook", "params", params)

	return webhook.ErrNotImplemented
}

func (h Handler) GetWebhook(ctx context.Context, params webhook.GetWebhookInput) (*webhook.Webhook, error) {
	h.logger.InfoContext(ctx, "getting webhook", "params", params)

	return nil, webhook.ErrNotImplemented
}

func (h Handler) ListWebhooks(ctx context.Context, params webhook.ListWebhooksInput) ([]webhook.Webhook, error) {
	h.logger.InfoContext(ctx, "listing webhooks", "params", params)

	return []webhook.Webhook{}, webhook.ErrNotImplemented
}

func (h Handler) SendMessage(ctx context.Context, params webhook.SendMessageInput) (*webhook.Message, error) {
	h.logger.InfoContext(ctx, "sending message", "params", params)

	return nil, webhook.ErrNotImplemented
}

func (h Handler) GetMessage(ctx context.Context, params webhook.GetMessageInput) (*webhook.Message, error) {
	h.logger.InfoContext(ctx, "getting message", "params", params)

	return nil, webhook.ErrNotImplemented
}

func (h Handler) ResendMessage(ctx context.Context, params webhook.ResendMessageInput) error {
	h.logger.InfoContext(ctx, "resending message", "params", params)

	return webhook.ErrNotImplemented
}
