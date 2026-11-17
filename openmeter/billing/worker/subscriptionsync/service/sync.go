package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	SubscriptionSyncComponentName billing.ComponentName = "subscription-sync"
	oldSubscriptionSyncThreshold                        = 30 * 24 * time.Hour

	// Subscription sync schedules its next reconciliation at a random point between
	// these percentages of the whole next billing period, measured from its start.
	nextReconciliationMinPercentage = 80
	nextReconciliationMaxPercentage = 95
)

func (s *Service) invoicePendingLines(ctx context.Context, customer customer.CustomerID) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.invoicePendingLines", trace.WithAttributes(
		attribute.String("customer_id", customer.ID),
	))

	return span.Wrap(func(ctx context.Context) error {
		_, err := s.billingService.InvoicePendingLines(
			ctx,
			billing.InvoicePendingLinesInput{
				Customer:          customer,
				ForceAsyncAdvance: s.forceAsyncInvoicePendingLines,
			},
			billing.WithPartialInvoiceLinesDisabled(),
			billing.WithMaxLinesPerInvoice(s.featureFlags.MaxLinesPerCollectedInvoice),
		)
		if err != nil {
			if errors.Is(err, billing.ErrInvoiceCreateNoLines) || errors.Is(err, billing.ErrInvoiceCreateAllLinesGated) {
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

	return s.SyncByID(ctx, event.Subscription.NamespacedID, time.Now(), subscriptionsync.WithInvoicePendingLines())
}

func (s *Service) SyncByID(ctx context.Context, subscriptionID models.NamespacedID, asOf time.Time, opts ...subscriptionsync.SynchronizeSubscriptionOption) error {
	options := subscriptionsync.SynchronizeSubscriptionOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	res, err := s.synchronizeSubscription(ctx, synchronizeSubscriptionInput{
		SubscriptionID: subscriptionID,
		AsOf:           asOf,
		Options:        options,
	})
	if err != nil {
		return fmt.Errorf("synchronize subscription: %w", err)
	}

	if options.InvoicePendingLines && !options.DryRun && res.ShouldInvoicePendingLines {
		// Collect after reconciliation so newly generated advance fees can be invoiced.
		if err := s.invoicePendingLines(ctx, res.CustomerID); err != nil {
			return fmt.Errorf("invoice pending lines (post): %w [customer_id=%s]", err, res.CustomerID.ID)
		}
	}

	return nil
}

type synchronizeSubscriptionResult struct {
	CustomerID                customer.CustomerID
	ShouldInvoicePendingLines bool
}

type synchronizeSubscriptionInput struct {
	SubscriptionID models.NamespacedID
	AsOf           time.Time
	Options        subscriptionsync.SynchronizeSubscriptionOptions
}

func (i synchronizeSubscriptionInput) Validate() error {
	var errs []error
	if err := i.SubscriptionID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("subscription ID: %w", err))
	}

	if i.AsOf.IsZero() {
		errs = append(errs, errors.New("as of is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type nextReconciliationAtInput struct {
	NextBillingPeriod    *timeutil.ClosedPeriod
	SubscriptionActiveTo *time.Time
	ReconciledAt         time.Time
	RandomFraction       float64
}

func (i nextReconciliationAtInput) Validate() error {
	var errs []error
	if i.NextBillingPeriod != nil {
		if err := i.NextBillingPeriod.ValidateAsRequired(); err != nil {
			errs = append(errs, fmt.Errorf("next billing period: %w", err))
		}
	} else if i.SubscriptionActiveTo == nil || i.SubscriptionActiveTo.IsZero() {
		errs = append(errs, errors.New("active to is required when there is no next billing period"))
	}

	if i.ReconciledAt.IsZero() {
		errs = append(errs, errors.New("reconciled at is required"))
	}

	if math.IsNaN(i.RandomFraction) || i.RandomFraction < 0 || i.RandomFraction > 1 {
		errs = append(errs, errors.New("random fraction must be between 0 and 1"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (s *Service) getNextReconciliationAt(input nextReconciliationAtInput) (time.Time, error) {
	if err := input.Validate(); err != nil {
		return time.Time{}, err
	}

	if input.NextBillingPeriod == nil {
		return *input.SubscriptionActiveTo, nil
	}

	percentage := nextReconciliationMinPercentage + (nextReconciliationMaxPercentage-nextReconciliationMinPercentage)*input.RandomFraction
	next := input.NextBillingPeriod.From.Add(time.Duration(float64(input.NextBillingPeriod.Duration()) * percentage / 100))
	// A late sync must not schedule itself in the past and be requeued on every poll.
	if !next.After(input.ReconciledAt) {
		return input.NextBillingPeriod.To, nil
	}

	return next, nil
}

func (s *Service) synchronizeSubscription(ctx context.Context, in synchronizeSubscriptionInput) (*synchronizeSubscriptionResult, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	subscriptionID := in.SubscriptionID
	asOf := in.AsOf
	options := in.Options
	span := tracex.Start[*synchronizeSubscriptionResult](ctx, s.tracer, "billing.worker.subscription.sync.SynchronizeSubscription", trace.WithAttributes(
		attribute.String("subscription_id", subscriptionID.ID),
		attribute.String("as_of", asOf.Format(time.RFC3339)),
	))

	return span.Wrap(func(ctx context.Context) (*synchronizeSubscriptionResult, error) {
		subscriptionForCustomer, err := s.getSubscription(ctx, subscriptionID)
		if err != nil {
			return nil, err
		}

		customerID := customer.CustomerID{
			Namespace: subscriptionForCustomer.Namespace,
			ID:        subscriptionForCustomer.CustomerId,
		}

		return withBillingLock(ctx, s, customerID, func(ctx context.Context) (*synchronizeSubscriptionResult, error) {
			syncStates, err := s.subscriptionSyncAdapter.GetSyncStatesV2(ctx, subscriptionsync.GetSyncStatesV2Input{subscriptionID})
			if err != nil {
				return nil, fmt.Errorf("getting subscription sync state v2: %w", err)
			}

			// Capture the revision before reading billing intent. A change committed
			// during sync must remain different from the revision acknowledged below.
			var subsRevision string
			if len(syncStates) == 0 {
				subsRevision = ulid.Make().String()
			} else {
				subsRevision = syncStates[0].CurrentSubsRevision
			}

			// A sync waiting for the billing lock must observe any deletion cleanup
			// that completed first, rather than recreate lines from an older view.
			subs, err := s.getSubscription(ctx, subscriptionID)
			if err != nil {
				return nil, err
			}

			customerProfile, err := s.billingService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
				Customer: customerID,
				Expand: billing.CustomerBillingProfileExpand{
					Customer: true,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("getting billing profile: %w", err)
			}

			if customerProfile.Customer == nil {
				return nil, errors.New("customer not found in billing profile")
			}

			customer := customerProfile.Customer

			var cleanupState subscriptionsync.SyncStateV2State
			switch {
			case subs.DeletedAt != nil:
				// Intentionally differs from IsDeleted(); see "Race between billing collection and subscription deletion cleanup" in ../README.md.
				cleanupState = subscriptionsync.SyncStateV2StateSubscriptionDeleted
			case customer.DeletedAt != nil && !subs.ActiveFrom.Before(*customer.DeletedAt):
				// This subscription cannot provide service to the deleted customer. Remove
				// any lines generated before deletion so billing collection cannot invoice them.
				cleanupState = subscriptionsync.SyncStateV2StateCustomerDeletedBefore
			}

			now := clock.Now().UTC()
			res := &synchronizeSubscriptionResult{CustomerID: customerID}
			// Avoid loading billing artifacts for subscriptions that ended more than 30 days ago.
			// Forced resyncs and deletion cleanup bypass this cutoff.
			if cleanupState == "" && !options.ForceResyncOldSubscriptions && subs.ActiveTo != nil && subs.ActiveTo.Before(now.Add(-oldSubscriptionSyncThreshold)) {
				s.logger.InfoContext(ctx, "skipping subscription sync because it ended more than 30 days ago", "subscription_id", subscriptionID.ID, "active_to", *subs.ActiveTo)
				if !options.DryRun {
					if err := s.subscriptionSyncAdapter.RecordSyncResultV2(ctx, subscriptionsync.RecordSyncResultV2Input{
						SubscriptionID:         subscriptionID,
						LastSuccessState:       subscriptionsync.SyncStateV2StateTooOld,
						LastSuccessAt:          clock.Now().UTC(),
						LastSyncedSubsRevision: subsRevision,
					}); err != nil {
						return nil, fmt.Errorf("recording skipped subscription sync: %w", err)
					}
				}

				return res, nil
			}

			var subsView *subscription.SubscriptionView
			if cleanupState == "" {
				view, err := s.subscriptionService.GetView(ctx, subscriptionID)
				if err != nil {
					return nil, fmt.Errorf("getting subscription view: %w", err)
				}

				subsView = &view
			}

			res.ShouldInvoicePendingLines = cleanupState == ""

			cur, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeFiat).
				WithCode(subs.InvoiceCurrency).
				Build()
			if err != nil {
				return nil, fmt.Errorf("getting currency calculator: %w", err)
			}

			// Calculate per line patches
			linesDiff, err := s.buildSyncPlan(ctx, buildSyncPlanInput{
				Subscription:                 subs,
				SubscriptionView:             subsView,
				AsOf:                         asOf,
				CustomerDeletedAt:            customer.DeletedAt,
				SubscriptionEndProrationMode: customerProfile.EffectiveProfile.WorkflowConfig.Invoicing.SubscriptionEndProrationMode,
				Currency:                     cur,
			})
			if err != nil {
				return nil, err
			}

			if linesDiff == nil {
				return nil, errors.New("building subscription sync plan returned nil without an error")
			}

			// An empty cleanup target must stop periodic scheduling even if the
			// subscription's spec still contains billable items.
			hasBillables := false
			if subsView != nil {
				hasBillables = subsView.Spec.HasBillables()
			}

			resultState := cleanupState
			if cleanupState == "" {
				resultState = subscriptionsync.SyncStateV2StateActive
				// Customer deletion can end billing before the subscription's own ActiveTo.
				// Record completion only once both the worker clock and AsOf cover that end.
				if billingEndsAt := linesDiff.BillingEndsAt; billingEndsAt != nil && !billingEndsAt.After(now) && !billingEndsAt.After(asOf) {
					resultState = subscriptionsync.SyncStateV2StateEnded
				} else if !hasBillables {
					resultState = subscriptionsync.SyncStateV2StateNoBillables
				}
			}

			if !linesDiff.IsEmpty() {
				if err := s.reconciler.Apply(ctx, reconciler.ApplyInput{
					DryRun:   options.DryRun,
					Customer: customerID,
					Currency: cur,
					Plan:     linesDiff,
				}); err != nil {
					return nil, err
				}
			}

			if options.DryRun {
				return res, nil
			}

			reconciledAt := clock.Now().UTC()
			var nextReconciliationAt *time.Time
			if resultState == subscriptionsync.SyncStateV2StateActive {
				next, err := s.getNextReconciliationAt(nextReconciliationAtInput{
					NextBillingPeriod:    linesDiff.NextBillingPeriod,
					SubscriptionActiveTo: linesDiff.BillingEndsAt,
					ReconciledAt:         reconciledAt,
					RandomFraction:       rand.Float64(),
				})
				if err != nil {
					return nil, fmt.Errorf("calculating next reconciliation at: %w", err)
				}

				if next.Before(reconciledAt) {
					s.logger.WarnContext(ctx, "next subscription reconciliation is in the past, which can cause repeated syncs",
						"subscription_id", subscriptionID.ID,
						"next_reconciliation_at", next,
						"reconciled_at", reconciledAt,
					)
				}

				nextReconciliationAt = &next
			}

			if err := s.updateSyncState(ctx, updateSyncStateInput{
				SubscriptionID:         subscriptionID,
				MaxGenerationTimeLimit: linesDiff.SubscriptionMaxGenerationTimeLimit,
				HasBillables:           hasBillables,
			}); err != nil {
				return nil, fmt.Errorf("updating sync state: %w", err)
			}

			// Record success even for an empty plan, together with billing changes and legacy scheduling.
			if err := s.subscriptionSyncAdapter.RecordSyncResultV2(ctx, subscriptionsync.RecordSyncResultV2Input{
				SubscriptionID:         subscriptionID,
				LastSuccessState:       resultState,
				LastSuccessAt:          reconciledAt,
				NextReconciliationAt:   nextReconciliationAt,
				LastSyncedSubsRevision: subsRevision,
			}); err != nil {
				return nil, fmt.Errorf("recording subscription sync result: %w", err)
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
	HasBillables           bool
}

func (s *Service) updateSyncState(ctx context.Context, in updateSyncStateInput) error {
	span := tracex.StartWithNoValue(ctx, s.tracer, "billing.worker.subscription.sync.updateSyncState", trace.WithAttributes(
		attribute.String("subscription_id", in.SubscriptionID.ID),
		attribute.String("max_generation_time_limit", in.MaxGenerationTimeLimit.Format(time.RFC3339)),
	))

	return span.Wrap(func(ctx context.Context) error {
		if !in.HasBillables {
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
