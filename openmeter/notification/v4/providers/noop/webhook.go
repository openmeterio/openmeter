package noop

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/pkg/models"
)

// WebhookProvider fills the registry's webhook slot when delivery is disabled.
// It has no resource state and never reports provisioning or delivery success.
type WebhookProvider struct {
	logger *slog.Logger
}

var _ v4.WebhookProvider = (*WebhookProvider)(nil)

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
func (*WebhookProvider) RoutingMode() v4.ProviderRoutingMode {
	return v4.ProviderRoutingModeExplicit
}

// GetNamespace reports disabled provisioning rather than inventing namespace state.
func (p *WebhookProvider) GetNamespace(ctx context.Context, input v4.GetNamespaceInput) (*v4.GetNamespaceResult, error) {
	return nil, p.operationError(ctx, input, "get namespace")
}

// CreateNamespace rejects provisioning without creating or retaining resources.
func (p *WebhookProvider) CreateNamespace(ctx context.Context, input v4.CreateNamespaceInput) (*v4.CreateNamespaceResult, error) {
	return nil, p.operationError(ctx, input, "create namespace")
}

// UpdateNamespace rejects updates without retaining supplied fields.
func (p *WebhookProvider) UpdateNamespace(ctx context.Context, input v4.UpdateNamespaceInput) (*v4.UpdateNamespaceResult, error) {
	return nil, p.operationError(ctx, input, "update namespace")
}

// DeleteNamespace does not claim to have removed resources owned by another provider.
func (p *WebhookProvider) DeleteNamespace(ctx context.Context, input v4.DeleteNamespaceInput) error {
	return p.operationError(ctx, input, "delete namespace")
}

// GetChannel reports disabled provisioning rather than fabricated convergence.
func (p *WebhookProvider) GetChannel(ctx context.Context, input v4.GetChannelInput) (*v4.GetChannelResult, error) {
	return nil, p.operationError(ctx, input, "get channel")
}

// CreateChannel rejects provisioning without contacting the configured webhook URL.
func (p *WebhookProvider) CreateChannel(ctx context.Context, input v4.CreateChannelInput[v4.WebhookConfig]) (*v4.CreateChannelResult, error) {
	return nil, p.operationError(ctx, input, "create channel")
}

// UpdateChannel rejects updates without contacting the configured webhook URL.
func (p *WebhookProvider) UpdateChannel(ctx context.Context, input v4.UpdateChannelInput[v4.WebhookConfigUpdate]) (*v4.UpdateChannelResult, error) {
	return nil, p.operationError(ctx, input, "update channel")
}

// DeleteChannel does not claim to have removed resources owned by another provider.
func (p *WebhookProvider) DeleteChannel(ctx context.Context, input v4.DeleteChannelInput) error {
	return p.operationError(ctx, input, "delete channel")
}

// SendEvent rejects delivery instead of silently accepting and dropping an Event.
func (p *WebhookProvider) SendEvent(ctx context.Context, input v4.SendEventInput) (*v4.SendEventResult, error) {
	return nil, p.operationError(ctx, input, "send event")
}

// GetEvent reports unavailable observations without fabricating delivery attempts.
func (p *WebhookProvider) GetEvent(ctx context.Context, input v4.GetEventInput) (*v4.GetEventResult, error) {
	return nil, p.operationError(ctx, input, "get event")
}

// ResendEvent rejects delivery without consuming or advancing dispatch generations.
func (p *WebhookProvider) ResendEvent(ctx context.Context, input v4.ResendEventInput) (*v4.ResendEventResult, error) {
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

	return fmt.Errorf("failed to %s: %w", operation, &v4.ProviderError{
		Kind: v4.ProviderErrorKindConfiguration,
		Code: v4.ProviderErrorCodeConfigurationInvalid,
		Err:  models.NewGenericNotImplementedError(errors.New("webhook provider is disabled")),
	})
}
