package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	SubscriptionSyncComponentName billing.ComponentName = "subscription-sync"
)

func (s *Service) invoicePendingLines(ctx context.Context, customer customer.CustomerID) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.invoicePendingLines", trace.WithAttributes(
		attribute.String("customer_id", customer.ID),
	))

	return span.Wrap(func(ctx context.Context) error {
		_, err := s.billingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer:                   customer,
			ProgressiveBillingOverride: lo.ToPtr(false),
		})
		if err != nil {
			if errors.Is(err, billing.ErrInvoiceCreateNoLines) {
				return nil
			}

			return err
		}

		return nil
	})
}

func (s *Service) HandleSubscriptionSyncEvent(ctx context.Context, event *subscription.SubscriptionSyncEvent) error {
	if event == nil {
		return nil
	}

	subsView, err := s.subscriptionService.GetView(ctx, event.Subscription.NamespacedID)
	if err != nil {
		return fmt.Errorf("getting subscription view: %w", err)
	}

	return s.SynchronizeSubscriptionAndInvoiceCustomer(ctx, subsView, time.Now())
}

func (s *Service) SynchronizeSubscriptionAndInvoiceCustomer(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.SynchronizeSubscriptionAndInvoiceCustomer", trace.WithAttributes(
		attribute.String("subscription_id", subs.Subscription.ID),
		attribute.String("as_of", asOf.Format(time.RFC3339)),
	))

	return span.Wrap(func(ctx context.Context) error {
		if err := s.SynchronizeSubscription(ctx, subs, asOf); err != nil {
			return fmt.Errorf("synchronize subscription: %w", err)
		}

		customerID := customer.CustomerID{
			Namespace: subs.Subscription.Namespace,
			ID:        subs.Subscription.CustomerId,
		}
		// Invoice any pending lines invoicable now, so that any in advance fees are invoiced immediately.
		if err := s.invoicePendingLines(ctx, customerID); err != nil {
			return fmt.Errorf("invoice pending lines (post): %w [customer_id=%s]", err, customerID.ID)
		}

		return nil
	})
}

func (s *Service) SynchronizeSubscription(ctx context.Context, subs subscription.SubscriptionView, asOf time.Time, opts ...subscriptionsync.SynchronizeSubscriptionOption) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.SynchronizeSubscription", trace.WithAttributes(
		attribute.String("subscription_id", subs.Subscription.ID),
		attribute.String("as_of", asOf.Format(time.RFC3339)),
	))

	options := subscriptionsync.SynchronizeSubscriptionOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return span.Wrap(func(ctx context.Context) error {
		if !subs.Spec.HasBillables() {
			if err := s.updateSyncState(ctx, updateSyncStateInput{
				SubscriptionView: subs,
			}); err != nil {
				return fmt.Errorf("updating sync state: %w", err)
			}

			s.logger.DebugContext(ctx, "subscription has no billables, skipping sync", "subscription_id", subs.Subscription.ID)
			return nil
		}

		customerID := customer.CustomerID{
			Namespace: subs.Subscription.Namespace,
			ID:        subs.Subscription.CustomerId,
		}

		// TODO[later]: Right now we are getting the billing profile as a validation step, but later if we allow more collection
		// alignment settings, we should use the collection settings from here to determine the generation end (overriding asof).
		_, err := s.billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customerID,
		})
		if err != nil {
			return fmt.Errorf("getting billing profile: %w", err)
		}

		currency, err := subs.Spec.Currency.Calculator()
		if err != nil {
			return fmt.Errorf("getting currency calculator: %w", err)
		}

		return s.billingService.WithLock(ctx, customer.CustomerID{
			Namespace: subs.Subscription.Namespace,
			ID:        subs.Subscription.CustomerId,
		}, func(ctx context.Context) error {
			persistedLoader := persistedstate.NewLoader(s.billingService)
			persistedInvoices, err := persistedLoader.LoadInvoicesForCustomer(ctx, customerID)
			if err != nil {
				return err
			}

			// Calculate per line patches
			linesDiff, err := s.buildSyncPlan(ctx, subs, asOf, currency)
			if err != nil {
				return err
			}

			if linesDiff == nil || linesDiff.IsEmpty() {
				generationLimit := time.Time{}
				if linesDiff != nil {
					generationLimit = linesDiff.SubscriptionMaxGenerationTimeLimit
				}

				if err := s.updateSyncState(ctx, updateSyncStateInput{
					SubscriptionView:       subs,
					MaxGenerationTimeLimit: generationLimit,
				}); err != nil {
					return fmt.Errorf("updating sync state: %w", err)
				}

				return nil
			}

			if err := s.reconciler.Apply(ctx, reconciler.ApplyInput{
				DryRun:       options.DryRun,
				Customer:     customerID,
				Subscription: subs,
				Currency:     currency,
				Invoices:     persistedInvoices,
				Plan:         linesDiff,
			}); err != nil {
				return err
			}

			if err := s.updateSyncState(ctx, updateSyncStateInput{
				SubscriptionView:       subs,
				MaxGenerationTimeLimit: linesDiff.SubscriptionMaxGenerationTimeLimit,
			}); err != nil {
				return fmt.Errorf("updating sync state: %w", err)
			}

			return nil
		})
	})
}

type updateSyncStateInput struct {
	SubscriptionView       subscription.SubscriptionView
	MaxGenerationTimeLimit time.Time
}

func (s *Service) updateSyncState(ctx context.Context, in updateSyncStateInput) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.updateSyncState", trace.WithAttributes(
		attribute.String("subscription_id", in.SubscriptionView.Subscription.ID),
		attribute.String("max_generation_time_limit", in.MaxGenerationTimeLimit.Format(time.RFC3339)),
	))

	return span.Wrap(func(ctx context.Context) error {
		if !in.SubscriptionView.Spec.HasBillables() {
			return s.subscriptionSyncAdapter.UpsertSyncState(ctx, subscriptionsync.UpsertSyncStateInput{
				SubscriptionID: models.NamespacedID{
					ID:        in.SubscriptionView.Subscription.ID,
					Namespace: in.SubscriptionView.Subscription.Namespace,
				},
				HasBillables: false,
				SyncedAt:     clock.Now().UTC(),
			})
		}

		nextSyncAfter := in.MaxGenerationTimeLimit

		if in.MaxGenerationTimeLimit.IsZero() {
			// Fallback: we cannot determine the next sync after, so we'll just mandate the sync
			if nextSyncAfter.IsZero() {
				s.logger.WarnContext(ctx, "cannot determine the next sync after, syncing immediately", "subscription_id", in.SubscriptionView.Subscription.ID)
				nextSyncAfter = clock.Now().UTC()
			}
		}

		return s.subscriptionSyncAdapter.UpsertSyncState(ctx, subscriptionsync.UpsertSyncStateInput{
			SubscriptionID: models.NamespacedID{
				ID:        in.SubscriptionView.Subscription.ID,
				Namespace: in.SubscriptionView.Subscription.Namespace,
			},
			HasBillables:  true,
			NextSyncAfter: lo.ToPtr(nextSyncAfter),
			SyncedAt:      clock.Now().UTC(),
		})
	})
}
