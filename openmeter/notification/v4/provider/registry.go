package provider

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

// RegistryConfig supplies a provider for every supported channel type.
type RegistryConfig struct {
	Logger  *slog.Logger
	Webhook WebhookProvider
}

var _ models.Validator = RegistryConfig{}

// Validate rejects incomplete wiring before providers can be accessed.
func (c RegistryConfig) Validate() error {
	var errs []error

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if lo.IsNil(c.Webhook) {
		errs = append(errs, errors.New("webhook provider is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Registry holds one provider per channel type, fixed at construction.
// Its zero value is not usable; construct it with NewRegistry.
// Application composition owns provider lifecycles.
type Registry struct {
	logger  *slog.Logger
	webhook WebhookProvider
}

// NewRegistry validates and captures all provider dependencies without
// creating external resources.
func NewRegistry(config RegistryConfig) (*Registry, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid provider registry configuration: %w", err)
	}

	return &Registry{
		logger:  config.Logger,
		webhook: config.Webhook,
	}, nil
}

// Webhook returns the webhook provider required at construction.
func (r *Registry) Webhook() WebhookProvider {
	return r.webhook
}
