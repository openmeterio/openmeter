package hooks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Adapter subscriptionsync.SyncStateAdapter
	Logger  *slog.Logger
}

func (c Config) Validate() error {
	var errs []error
	if c.Adapter == nil {
		errs = append(errs, errors.New("subscription sync adapter is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Hook struct {
	subscription.NoOpSubscriptionCommandHook
	adapter subscriptionsync.SyncStateAdapter
	logger  *slog.Logger
}

var _ subscription.SubscriptionCommandHook = (*Hook)(nil)

func New(config Config) (*Hook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validating subscription sync hook: %w", err)
	}

	return &Hook{adapter: config.Adapter, logger: config.Logger}, nil
}

func (h *Hook) AfterCreate(ctx context.Context, view subscription.SubscriptionView) error {
	return h.markChanged(ctx, view.Subscription.NamespacedID)
}

func (h *Hook) AfterUpdate(ctx context.Context, view subscription.SubscriptionView) error {
	return h.markChanged(ctx, view.Subscription.NamespacedID)
}

func (h *Hook) AfterCancel(ctx context.Context, view subscription.SubscriptionView) error {
	return h.markChanged(ctx, view.Subscription.NamespacedID)
}

func (h *Hook) AfterContinue(ctx context.Context, view subscription.SubscriptionView) error {
	return h.markChanged(ctx, view.Subscription.NamespacedID)
}

func (h *Hook) AfterDelete(ctx context.Context, subscriptionID models.NamespacedID) error {
	return h.markChanged(ctx, subscriptionID)
}

func (h *Hook) markChanged(ctx context.Context, subscriptionID models.NamespacedID) error {
	if err := h.adapter.MarkSubscriptionChanged(ctx, subscriptionID); err != nil {
		return fmt.Errorf("recording subscription change for sync: %w", err)
	}

	h.logger.DebugContext(ctx, "marked subscription changed for sync", "subscription_id", subscriptionID.ID)

	return nil
}
