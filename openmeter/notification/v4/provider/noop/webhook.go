package noop

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	notificationprovider "github.com/openmeterio/openmeter/openmeter/notification/v4/provider"
	"github.com/openmeterio/openmeter/pkg/models"
)

// WebhookProvider fills the registry's webhook slot when delivery is disabled.
// It has no resource state and never reports provisioning or delivery success.
type WebhookProvider struct {
	logger *slog.Logger
}

var _ notificationprovider.WebhookProvider = (*WebhookProvider)(nil)

// NewWebhookProvider constructs a disabled provider without external dependencies.
func NewWebhookProvider(logger *slog.Logger) (*WebhookProvider, error) {
	if logger == nil {
		return nil, models.NewGenericValidationError(errors.New("invalid logger: required"))
	}

	return &WebhookProvider{
		logger: logger.With("notification.provider", "noop"),
	}, nil
}

// RoutingMode selects explicit destinations; the no-op provider never routes or sends them.
func (*WebhookProvider) RoutingMode() notificationprovider.ProviderRoutingMode {
	return notificationprovider.ProviderRoutingModeExplicit
}

// GetNamespace reports disabled provisioning rather than inventing namespace state.
func (p *WebhookProvider) GetNamespace(ctx context.Context, input notificationprovider.GetNamespaceInput) (*notificationprovider.GetNamespaceResult, error) {
	return nil, p.operationError(ctx, input, "get namespace")
}

// CreateNamespace rejects provisioning without creating or retaining resources.
func (p *WebhookProvider) CreateNamespace(ctx context.Context, input notificationprovider.CreateNamespaceInput) (*notificationprovider.CreateNamespaceResult, error) {
	return nil, p.operationError(ctx, input, "create namespace")
}

// UpdateNamespace rejects updates without retaining supplied fields.
func (p *WebhookProvider) UpdateNamespace(ctx context.Context, input notificationprovider.UpdateNamespaceInput) (*notificationprovider.UpdateNamespaceResult, error) {
	return nil, p.operationError(ctx, input, "update namespace")
}

// DeleteNamespace does not claim to have removed resources owned by another provider.
func (p *WebhookProvider) DeleteNamespace(ctx context.Context, input notificationprovider.DeleteNamespaceInput) error {
	return p.operationError(ctx, input, "delete namespace")
}

// GetChannel reports disabled provisioning rather than fabricated convergence.
func (p *WebhookProvider) GetChannel(ctx context.Context, input notificationprovider.GetChannelInput) (*notificationprovider.GetChannelResult, error) {
	return nil, p.operationError(ctx, input, "get channel")
}

// CreateChannel rejects provisioning without contacting the configured webhook URL.
func (p *WebhookProvider) CreateChannel(ctx context.Context, input notificationprovider.CreateChannelInput[v4.WebhookConfig]) (*notificationprovider.CreateChannelResult, error) {
	return nil, p.operationError(ctx, input, "create channel")
}

// UpdateChannel rejects updates without contacting the configured webhook URL.
func (p *WebhookProvider) UpdateChannel(ctx context.Context, input notificationprovider.UpdateChannelInput[v4.WebhookConfigUpdate]) (*notificationprovider.UpdateChannelResult, error) {
	return nil, p.operationError(ctx, input, "update channel")
}

// DeleteChannel does not claim to have removed resources owned by another provider.
func (p *WebhookProvider) DeleteChannel(ctx context.Context, input notificationprovider.DeleteChannelInput) error {
	return p.operationError(ctx, input, "delete channel")
}

// SendEvent rejects delivery instead of silently accepting and dropping an Event.
func (p *WebhookProvider) SendEvent(ctx context.Context, input notificationprovider.SendEventInput) (*notificationprovider.SendEventResult, error) {
	return nil, p.operationError(ctx, input, "send event")
}

// GetEvent reports unavailable observations without fabricating delivery attempts.
func (p *WebhookProvider) GetEvent(ctx context.Context, input notificationprovider.GetEventInput) (*notificationprovider.GetEventResult, error) {
	return nil, p.operationError(ctx, input, "get event")
}

// ResendEvent rejects delivery without consuming or advancing dispatch generations.
func (p *WebhookProvider) ResendEvent(ctx context.Context, input notificationprovider.ResendEventInput) (*notificationprovider.ResendEventResult, error) {
	return nil, p.operationError(ctx, input, "resend event")
}

// operationError keeps caller cancellation and malformed inputs distinct from a
// disabled integration. Inputs are never logged because they may contain secrets.
func (p *WebhookProvider) operationError(ctx context.Context, input models.Validator, operation string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("failed to %s: %w", operation, err)
	}

	if err := input.Validate(); err != nil {
		return fmt.Errorf("invalid %s input: %w", operation, err)
	}

	p.logger.DebugContext(ctx, "webhook provider is disabled", "operation", operation)

	return fmt.Errorf("failed to %s: %w", operation, &notificationprovider.ProviderError{
		Kind: notificationprovider.ProviderErrorKindConfiguration,
		Code: notificationprovider.ProviderErrorCodeConfigurationInvalid,
		Err:  models.NewGenericNotImplementedError(errors.New("webhook provider is disabled")),
	})
}
