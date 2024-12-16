package webhook

import (
	"context"
	"log/slog"
	"time"
)

var _ Handler = (*debugWebhookHandler)(nil)

// debugWebhookHandler logs all operations but doesn't actually handle webhooks.
type debugWebhookHandler struct {
	logger *slog.Logger
}

func newDebugWebhookHandler(logger *slog.Logger) Handler {
	return &debugWebhookHandler{
		logger: logger.WithGroup("webhook"),
	}
}

func (h debugWebhookHandler) RegisterEventTypes(ctx context.Context, params RegisterEventTypesInputs) error {
	for _, eventType := range params.EventTypes {
		h.logger.InfoContext(ctx, "registering event types", "event_type", eventType)
	}

	return nil
}

func (h debugWebhookHandler) CreateWebhook(ctx context.Context, params CreateWebhookInput) (*Webhook, error) {
	h.logger.InfoContext(ctx, "creating webhook", "params", params)

	return &Webhook{
		ID:         "webhook-id",
		URL:        params.URL,
		Secret:     "webhook-secret",
		Disabled:   params.Disabled,
		RateLimit:  params.RateLimit,
		EventTypes: params.EventTypes,
		Channels:   params.Channels,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

func (h debugWebhookHandler) UpdateWebhook(ctx context.Context, params UpdateWebhookInput) (*Webhook, error) {
	h.logger.InfoContext(ctx, "updating webhook", "params", params)

	return &Webhook{
		ID:         "webhook-id",
		URL:        params.URL,
		Secret:     "webhook-secret",
		Disabled:   params.Disabled,
		RateLimit:  params.RateLimit,
		EventTypes: params.EventTypes,
		Channels:   params.Channels,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

func (h debugWebhookHandler) UpdateWebhookChannels(ctx context.Context, params UpdateWebhookChannelsInput) (*Webhook, error) {
	h.logger.InfoContext(ctx, "updating webhook channels", "params", params)

	return &Webhook{
		ID:        "webhook-id",
		Secret:    "webhook-secret",
		Channels:  params.AddChannels,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (h debugWebhookHandler) DeleteWebhook(ctx context.Context, params DeleteWebhookInput) error {
	h.logger.InfoContext(ctx, "deleting webhook", "params", params)

	return nil
}

func (h debugWebhookHandler) GetWebhook(ctx context.Context, params GetWebhookInput) (*Webhook, error) {
	h.logger.InfoContext(ctx, "getting webhook", "params", params)

	return &Webhook{
		ID: "webhook-id",
	}, nil
}

func (h debugWebhookHandler) ListWebhooks(ctx context.Context, params ListWebhooksInput) ([]Webhook, error) {
	h.logger.InfoContext(ctx, "listing webhooks", "params", params)

	return []Webhook{
		{
			ID: "webhook-id",
		},
	}, nil
}

func (h debugWebhookHandler) SendMessage(ctx context.Context, params SendMessageInput) (*Message, error) {
	h.logger.InfoContext(ctx, "sending message", "params", params)

	return &Message{
		ID: "message-id",
	}, nil
}
