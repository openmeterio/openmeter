package v4

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ProviderRegistryConfig supplies a provider for every supported channel type.
type ProviderRegistryConfig struct {
	Logger  *slog.Logger
	Webhook WebhookProvider
}

var _ models.Validator = ProviderRegistryConfig{}

// Validate rejects incomplete wiring before providers can be accessed.
func (c ProviderRegistryConfig) Validate() error {
	var errs []error

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if lo.IsNil(c.Webhook) {
		errs = append(errs, errors.New("webhook provider is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ProviderRegistry holds one provider per channel type, fixed at construction.
// Its zero value is not usable; construct it with NewProviderRegistry.
// Application composition owns provider lifecycles.
type ProviderRegistry struct {
	logger  *slog.Logger
	webhook WebhookProvider
}

// NewProviderRegistry validates and captures all provider dependencies without
// creating external resources.
func NewProviderRegistry(config ProviderRegistryConfig) (*ProviderRegistry, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid provider registry configuration: %w", err)
	}

	return &ProviderRegistry{
		logger:  config.Logger,
		webhook: config.Webhook,
	}, nil
}

// Webhook returns the webhook provider required at construction.
func (r *ProviderRegistry) Webhook() WebhookProvider {
	return r.webhook
}
