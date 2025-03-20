package billingworkersubscription

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Reconciler is a component that periodically reconciles the subscription state with the billing state
// this is essential, as the invoice creation is purley event driven. In case a processing error happens,
// we might fail to create an invoice, and stop processing the subscription.
type Reconciler struct {
	subscriptionSync    *Handler
	subscriptionService subscription.Service

	logger *slog.Logger
}

type ReconcilerConfig struct {
	SubscriptionSync    *Handler
	SubscriptionService subscription.Service

	Logger *slog.Logger
}

func (c ReconcilerConfig) Validate() error {
	if c.SubscriptionSync == nil {
		return errors.New("subscriptionSync is required")
	}

	if c.SubscriptionService == nil {
		return errors.New("subscriptionService is required")
	}

	if c.Logger == nil {
		return errors.New("logger is required")
	}

	return nil
}

func NewReconciler(config ReconcilerConfig) (*Reconciler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Reconciler{
		subscriptionSync:    config.SubscriptionSync,
		subscriptionService: config.SubscriptionService,
		logger:              config.Logger,
	}, nil
}

type ReconcilerListSubscriptionsInput struct {
	Namespaces []string
	Customers  []string
	Lookback   time.Duration
}

func (i ReconcilerListSubscriptionsInput) Validate() error {
	if i.Lookback <= 0 {
		return errors.New("lookback must be greater than 0")
	}

	return nil
}

func (r *Reconciler) ListSubscriptions(ctx context.Context, in ReconcilerListSubscriptionsInput) ([]subscription.Subscription, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	subscriptions, err := r.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces: in.Namespaces,
		Customers:  in.Customers,
		ActiveAt:   lo.ToPtr(time.Now().Add(-in.Lookback)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	return subscriptions.Items, nil
}

func (r *Reconciler) ReconcileSubscription(ctx context.Context, subsID models.NamespacedID) error {
	subsView, err := r.subscriptionService.GetView(ctx, subsID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	return r.subscriptionSync.SyncronizeSubscription(ctx, subsView, time.Now())
}

type ReconcilerAllInput = ReconcilerListSubscriptionsInput

func (r *Reconciler) All(ctx context.Context, in ReconcilerAllInput) error {
	subscriptions, err := r.ListSubscriptions(ctx, in)
	if err != nil {
		return fmt.Errorf("failed to list subscriptions: %w", err)
	}

	var outErr error
	for _, subscription := range subscriptions {
		if err := r.ReconcileSubscription(ctx, subscription.NamespacedID); err != nil {
			r.logger.ErrorContext(ctx, "failed to reconcile subscription", "error", err)

			outErr = errors.Join(outErr, fmt.Errorf("failed to reconcile subscription: %w", err))
		}
	}

	return outErr
}
