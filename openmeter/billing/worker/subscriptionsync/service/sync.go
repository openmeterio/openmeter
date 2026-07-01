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
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

const (
	SubscriptionSyncComponentName billing.ComponentName = "subscription-sync"
)

const (
	maxInvoicePendingCollectionPasses = 5
)

func (s *Service) invoicePendingLines(ctx context.Context, customer customer.CustomerID) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.invoicePendingLines", trace.WithAttributes(
		attribute.String("customer_id", customer.ID),
	))

	return span.Wrap(func(ctx context.Context) error {
		for pass := 0; pass < maxInvoicePendingCollectionPasses; pass++ {
			invoices, err := s.billingService.InvoicePendingLines(
				ctx,
				billing.InvoicePendingLinesInput{
					Customer: customer,
				},
				billing.WithPartialInvoiceLinesDisabled(),
				billing.WithMaxLinesPerInvoice(s.featureFlags.MaxLinesPerCollectedInvoice),
			)
			if err != nil {
				if errors.Is(err, billing.ErrInvoiceCreateNoLines) {
					return nil
				}

				return err
			}

			if s.featureFlags.MaxLinesPerCollectedInvoice <= 0 || len(invoices) == 0 {
				return nil
			}
		}

		return fmt.Errorf("exceeded max invoice pending collection passes for customer %s", customer.ID)
	})
}

func (s *Service) HandleSubscriptionSyncEvent(ctx context.Context, event *subscription.SubscriptionSyncEvent) error {
	if event == nil {
		return nil
	}

	return s.synchronizeSubscriptionAndInvoiceCustomer(ctx, newSubscriptionReferenceOrView(event.Subscription), time.Now())
}

func (s *Service) synchronizeSubscriptionAndInvoiceCustomer(ctx context.Context, refOrView subscriptionReferenceOrView, asOf time.Time) error {
	res, err := s.synchronizeSubscription(ctx, refOrView, asOf)
	if err != nil {
		return fmt.Errorf("synchronize subscription: %w", err)
	}

	if res != nil && res.View != nil {
		customerID := customer.CustomerID{
			Namespace: res.View.Subscription.Namespace,
			ID:        res.View.Subscription.CustomerId,
		}
		// Invoice any pending lines invoicable now, so that any in advance fees are invoiced immediately.
		if err := s.invoicePendingLines(ctx, customerID); err != nil {
			return fmt.Errorf("invoice pending lines (post): %w [customer_id=%s]", err, customerID.ID)
		}
	}

	return nil
}

type synchronizeSubscriptionResult struct {
	View    *subscription.SubscriptionView
	Deleted bool
}

func (s *Service) synchronizeSubscription(ctx context.Context, refOrView subscriptionReferenceOrView, asOf time.Time, opts ...subscriptionsync.SynchronizeSubscriptionOption) (*synchronizeSubscriptionResult, error) {
	subscriptionID := refOrView.GetID()

	span := tracex.Start[*synchronizeSubscriptionResult](ctx, s.tracer, "billing.worker.subscription.sync.SynchronizeSubscription", trace.WithAttributes(
		attribute.String("subscription_id", subscriptionID.ID),
		attribute.String("as_of", asOf.Format(time.RFC3339)),
	))

	options := subscriptionsync.SynchronizeSubscriptionOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return span.Wrap(func(ctx context.Context) (*synchronizeSubscriptionResult, error) {
		subs, err := s.getSubscription(ctx, subscriptionID)
		if err != nil {
			return nil, err
		}

		var subsView *subscription.SubscriptionView
		if subs.IsDeleted() {
			subsView = nil
		} else if refOrView.Type() == SubscriptionReferenceTypeView {
			view, err := refOrView.AsSubscriptionView()
			if err != nil {
				return nil, err
			}

			subsView = &view
		} else {
			view, err := s.subscriptionService.GetView(ctx, subscriptionID)
			if err != nil {
				return nil, err
			}

			subsView = &view
		}

		res := &synchronizeSubscriptionResult{
			View:    subsView,
			Deleted: subs.IsDeleted(),
		}

		customerID := customer.CustomerID{
			Namespace: subs.Namespace,
			ID:        subs.CustomerId,
		}

		var customerDeletedAt *time.Time
		var subscriptionEndProrationMode billing.SubscriptionEndProrationMode
		if subsView != nil && subsView.Spec.HasBillables() {
			// TODO[later]: Right now we are getting the billing profile as a validation step, but later if we allow more collection
			// alignment settings, we should use the collection settings from here to determine the generation end (overriding asof).
			customerOverride, err := s.billingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
				Customer: customerID,
				Expand: billing.CustomerOverrideExpand{
					Customer: true,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("getting billing profile: %w", err)
			}

			if customerOverride.Customer != nil {
				customerDeletedAt = convert.SafeToUTC(customerOverride.Customer.GetDeletedAt())
			}
			subscriptionEndProrationMode = customerOverride.MergedProfile.WorkflowConfig.Invoicing.SubscriptionEndProrationMode

			if customerOverride.Customer != nil && customerOverride.Customer.DeletedAt != nil && !customerOverride.Customer.DeletedAt.After(subsView.Spec.ActiveFrom) {
				if options.DryRun {
					return res, nil
				}

				if err := s.updateSyncState(ctx, updateSyncStateInput{
					SubscriptionID: subscriptionID,
					// Prevent deleted customers from continuing to be scheduled for sync.
					PreventFurtherSyncs: true,
				}); err != nil {
					return nil, fmt.Errorf("updating sync state: %w", err)
				}

				s.logger.WarnContext(ctx, "customer deleted before subscription start, skipping sync", "subscription_id", subscriptionID.ID, "customer_id", customerID.ID)
				return res, nil
			}
		}

		currency, err := subs.Currency.Calculator()
		if err != nil {
			return nil, fmt.Errorf("getting currency calculator: %w", err)
		}

		return withBillingLock(ctx, s, customer.CustomerID{
			Namespace: subs.Namespace,
			ID:        subs.CustomerId,
		}, func(ctx context.Context) (*synchronizeSubscriptionResult, error) {
			// Calculate per line patches
			linesDiff, err := s.buildSyncPlan(ctx, buildSyncPlanInput{
				Subscription:                 subs,
				SubscriptionView:             subsView,
				AsOf:                         asOf,
				CustomerDeletedAt:            customerDeletedAt,
				SubscriptionEndProrationMode: subscriptionEndProrationMode,
				Currency:                     currency,
				DryRun:                       options.DryRun,
			})
			if err != nil {
				return nil, err
			}

			// If we have a view, we can use it to determine if the subscription has billables, if the
			// subscription is deleted (no view), let's set hasBillables to false, so that we prevent further syncs.
			hasBillables := false
			if subsView != nil {
				hasBillables = subsView.Spec.HasBillables()
			}

			if linesDiff == nil || linesDiff.IsEmpty() {
				if options.DryRun {
					return res, nil
				}

				generationLimit := time.Time{}
				if linesDiff != nil {
					generationLimit = linesDiff.SubscriptionMaxGenerationTimeLimit
				}

				if err := s.updateSyncState(ctx, updateSyncStateInput{
					SubscriptionID:         subscriptionID,
					MaxGenerationTimeLimit: generationLimit,
					HasBillables:           hasBillables,
				}); err != nil {
					return nil, fmt.Errorf("updating sync state: %w", err)
				}

				return res, nil
			}

			if err := s.reconciler.Apply(ctx, reconciler.ApplyInput{
				DryRun:   options.DryRun,
				Customer: customerID,
				Currency: currency,
				Plan:     linesDiff,
			}); err != nil {
				return nil, err
			}

			if options.DryRun {
				return res, nil
			}

			if err := s.updateSyncState(ctx, updateSyncStateInput{
				SubscriptionID:         subscriptionID,
				MaxGenerationTimeLimit: linesDiff.SubscriptionMaxGenerationTimeLimit,
				HasBillables:           hasBillables,
			}); err != nil {
				return nil, fmt.Errorf("updating sync state: %w", err)
			}

			return res, nil
		})
	})
}

func withBillingLock[T any](ctx context.Context, s *Service, customerID customer.CustomerID, fn func(ctx context.Context) (T, error)) (T, error) {
	var out T
	err := s.billingService.WithLock(ctx, customerID, func(ctx context.Context) error {
		var err error
		out, err = fn(ctx)
		return err
	})
	if err != nil {
		return lo.Empty[T](), err
	}

	return out, nil
}

func (s *Service) getSubscription(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	subs, err := s.subscriptionService.List(ctx, subscription.ListSubscriptionsInput{
		Namespaces:     []string{subscriptionID.Namespace},
		ID:             &filter.FilterULID{FilterString: filter.FilterString{In: &[]string{subscriptionID.ID}}},
		IncludeDeleted: true,
		Page: pagination.Page{
			PageNumber: 1,
			PageSize:   1,
		},
	})
	if err != nil {
		return subscription.Subscription{}, fmt.Errorf("getting subscription: %w", err)
	}

	if len(subs.Items) == 0 {
		return subscription.Subscription{}, subscription.NewSubscriptionNotFoundError(subscriptionID.ID)
	}

	return subs.Items[0], nil
}

type updateSyncStateInput struct {
	SubscriptionID         models.NamespacedID
	MaxGenerationTimeLimit time.Time
	PreventFurtherSyncs    bool
	HasBillables           bool
}

func (s *Service) updateSyncState(ctx context.Context, in updateSyncStateInput) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.updateSyncState", trace.WithAttributes(
		attribute.String("subscription_id", in.SubscriptionID.ID),
		attribute.String("max_generation_time_limit", in.MaxGenerationTimeLimit.Format(time.RFC3339)),
	))

	return span.Wrap(func(ctx context.Context) error {
		hasBillables := in.HasBillables
		if in.PreventFurtherSyncs {
			// We are using the hasBillables flag to prevent further syncs, this is used when the customer is
			// deleted etc.
			hasBillables = false
		}

		if !hasBillables {
			return s.subscriptionSyncAdapter.UpsertSyncState(ctx, subscriptionsync.UpsertSyncStateInput{
				SubscriptionID: in.SubscriptionID,
				HasBillables:   false,
				SyncedAt:       clock.Now().UTC(),
			})
		}

		nextSyncAfter := in.MaxGenerationTimeLimit

		if in.MaxGenerationTimeLimit.IsZero() {
			// Fallback: we cannot determine the next sync after, so we'll just mandate the sync
			if nextSyncAfter.IsZero() {
				s.logger.WarnContext(ctx, "cannot determine the next sync after, syncing immediately", "subscription_id", in.SubscriptionID.ID)
				nextSyncAfter = clock.Now().UTC()
			}
		}

		return s.subscriptionSyncAdapter.UpsertSyncState(ctx, subscriptionsync.UpsertSyncStateInput{
			SubscriptionID: in.SubscriptionID,
			HasBillables:   true,
			NextSyncAfter:  lo.ToPtr(nextSyncAfter),
			SyncedAt:       clock.Now().UTC(),
		})
	})
}
