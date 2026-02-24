package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionvalidators "github.com/openmeterio/openmeter/openmeter/subscription/validators/subscription"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/ffx"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type ServiceConfig struct {
	SubscriptionRepo      subscription.SubscriptionRepository
	SubscriptionPhaseRepo subscription.SubscriptionPhaseRepository
	SubscriptionItemRepo  subscription.SubscriptionItemRepository
	// connectors
	CustomerService customer.Service
	FeatureService  feature.FeatureConnector
	// adapters
	EntitlementAdapter subscription.EntitlementAdapter
	// framework
	TransactionManager transaction.Creator
	Publisher          eventbus.Publisher
	Lockr              *lockr.Locker
	FeatureFlags       ffx.Service

	// Hooks
	Hooks []subscription.SubscriptionCommandHook
}

func New(conf ServiceConfig) (subscription.Service, error) {
	svc := &service{
		ServiceConfig: conf,
	}

	val, err := subscriptionvalidators.NewSubscriptionUniqueConstraintValidator(subscriptionvalidators.SubscriptionUniqueConstraintValidatorConfig{
		FeatureFlags:    conf.FeatureFlags,
		QueryService:    svc,
		CustomerService: svc.CustomerService,
	})
	if err != nil {
		return nil, err
	}

	if err := svc.RegisterHook(val); err != nil {
		return nil, err
	}

	return svc, nil
}

var _ subscription.Service = &service{}

type service struct {
	ServiceConfig

	mu sync.RWMutex
}

func (s *service) RegisterHook(validator subscription.SubscriptionCommandHook) error {
	if validator == nil {
		return errors.New("invalid subscription validator: nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.Hooks = append(s.Hooks, validator)

	return nil
}

func (s *service) lockCustomer(ctx context.Context, customerId string) error {
	key, err := subscription.GetCustomerLock(customerId)
	if err != nil {
		return fmt.Errorf("failed to get customer lock: %w", err)
	}

	if err := s.Lockr.LockForTX(ctx, key); err != nil {
		return fmt.Errorf("failed to lock customer: %w", err)
	}

	return nil
}

func (s *service) Create(ctx context.Context, namespace string, spec subscription.SubscriptionSpec) (subscription.Subscription, error) {
	ctx = subscription.NewSubscriptionOperationContext(ctx)
	setSpanAttrs(ctx,
		attribute.String("subscription.namespace", namespace),
		attribute.String("subscription.operation", "create"),
	)
	setSpanAttrs(ctx, addSpecAttrs([]attribute.KeyValue{}, "subscription.spec.input", spec)...)

	def := subscription.Subscription{}

	// Fetch the customer & validate the customer
	cus, err := s.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: namespace,
			ID:        spec.CustomerId,
		},
	})
	if err != nil {
		return def, err
	}

	if cus != nil && cus.IsDeleted() {
		return def, models.NewGenericPreConditionFailedError(
			fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
		)
	}

	if cus == nil {
		return def, fmt.Errorf("customer is nil")
	}

	if err := s.validateCreate(ctx, *cus, spec); err != nil {
		return def, err
	}

	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
		if err := s.lockCustomer(ctx, spec.CustomerId); err != nil {
			return def, err
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		err = errors.Join(lo.Map(s.Hooks, func(v subscription.SubscriptionCommandHook, _ int) error {
			return v.BeforeCreate(ctx, namespace, spec)
		})...)
		if err != nil {
			return def, fmt.Errorf("failed to validate subscription: %w", err)
		}

		// Create subscription entity
		sub, err := s.SubscriptionRepo.Create(ctx, spec.ToCreateSubscriptionEntityInput(namespace))
		if err != nil {
			return def, fmt.Errorf("failed to create subscription: %w", err)
		}

		for _, phase := range spec.GetSortedPhases() {
			// Sanity check
			if phase == nil {
				return def, fmt.Errorf("nil phase")
			}

			phaseCadence, err := spec.GetPhaseCadence(phase.PhaseKey)
			if err != nil {
				return def, fmt.Errorf("failed to get phase cadence: %w", err)
			}

			if _, err := s.createPhase(ctx, *cus, *phase, sub, phaseCadence); err != nil {
				return def, err
			}
		}

		// Let's update the customer currency if needed
		if err := s.updateCustomerCurrencyIfNotSet(ctx, sub, spec); err != nil {
			return def, err
		}

		// Let's fetch the view for event generation
		view, err := s.GetView(ctx, sub.NamespacedID)
		if err != nil {
			return sub, err
		}

		err = errors.Join(lo.Map(s.Hooks, func(v subscription.SubscriptionCommandHook, _ int) error {
			return v.AfterCreate(ctx, view)
		})...)
		if err != nil {
			return sub, fmt.Errorf("failed to validate subscription: %w", err)
		}

		err = s.Publisher.Publish(ctx, subscription.NewCreatedEvent(ctx, view))
		if err != nil {
			return sub, fmt.Errorf("failed to publish event: %w", err)
		}

		// Return sub reference
		return sub, nil
	})
}

func (s *service) Update(ctx context.Context, subscriptionID models.NamespacedID, newSpec subscription.SubscriptionSpec) (subscription.Subscription, error) {
	ctx = subscription.NewSubscriptionOperationContext(ctx)
	setSpanAttrs(ctx,
		attribute.String("subscription.namespace", subscriptionID.Namespace),
		attribute.String("subscription.id", subscriptionID.ID),
		attribute.String("subscription.operation", "update"),
	)
	setSpanAttrs(ctx, addSpecAttrs([]attribute.KeyValue{}, "subscription.spec.input", newSpec)...)

	var def subscription.Subscription

	// Get the full view
	view, err := s.GetView(ctx, subscriptionID)
	if err != nil {
		return def, fmt.Errorf("failed to get view: %w", err)
	}
	setSpanAttrs(ctx, addViewAttrs([]attribute.KeyValue{}, "subscription.view.current", view)...)
	setSpanAttrs(ctx, addSpecAttrs([]attribute.KeyValue{}, "subscription.spec.current", view.Spec)...)

	if err := s.validateUpdate(ctx, view, newSpec); err != nil {
		return def, err
	}

	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		err = errors.Join(lo.Map(s.Hooks, func(v subscription.SubscriptionCommandHook, _ int) error {
			return v.BeforeUpdate(ctx, subscriptionID, newSpec)
		})...)
		if err != nil {
			return def, fmt.Errorf("failed to validate subscription: %w", err)
		}

		subs, err := s.sync(ctx, view, newSpec)
		if err != nil {
			return subs, err
		}

		// Let's fetch the view for event generation
		updatedView, err := s.GetView(ctx, subs.NamespacedID)
		if err != nil {
			return subs, err
		}
		setSpanAttrs(ctx, addViewAttrs([]attribute.KeyValue{}, "subscription.view.updated", updatedView)...)
		setSpanAttrs(ctx, addSpecAttrs([]attribute.KeyValue{}, "subscription.spec.updated", updatedView.Spec)...)

		err = errors.Join(lo.Map(s.Hooks, func(v subscription.SubscriptionCommandHook, _ int) error {
			return v.AfterUpdate(ctx, updatedView)
		})...)
		if err != nil {
			return subs, fmt.Errorf("failed to validate subscription: %w", err)
		}

		err = s.Publisher.Publish(ctx, subscription.NewUpdatedEvent(ctx, updatedView))
		if err != nil {
			return subs, fmt.Errorf("failed to publish event: %w", err)
		}

		return subs, nil
	})
}

func (s *service) Delete(ctx context.Context, subscriptionID models.NamespacedID) error {
	ctx = subscription.NewSubscriptionOperationContext(ctx)

	currentTime := clock.Now()

	// First, let's get the subscription
	view, err := s.GetView(ctx, subscriptionID)
	if err != nil {
		return err
	}

	// Let's make sure Delete is possible based on the transition rules
	if err := subscription.NewStateMachine(
		view.Subscription.GetStatusAt(currentTime),
	).CanTransitionOrErr(ctx, subscription.SubscriptionActionDelete); err != nil {
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	err = errors.Join(lo.Map(s.Hooks, func(v subscription.SubscriptionCommandHook, _ int) error {
		return v.BeforeDelete(ctx, view)
	})...)
	if err != nil {
		return fmt.Errorf("failed to validate subscription: %w", err)
	}

	return transaction.RunWithNoValue(ctx, s.TransactionManager, func(ctx context.Context) error {
		// First, let's delete all phases
		for _, phase := range view.Phases {
			if err := s.deletePhase(ctx, phase); err != nil {
				return fmt.Errorf("failed to delete phase: %w", err)
			}
		}

		// Then let's delete the subscription itself
		if err := s.SubscriptionRepo.Delete(ctx, view.Subscription.NamespacedID); err != nil {
			return fmt.Errorf("failed to delete subscription: %w", err)
		}

		// Let's publish the event for the deletion
		err = s.Publisher.Publish(ctx, subscription.NewDeletedEvent(ctx, view))
		if err != nil {
			return fmt.Errorf("failed to publish event: %w", err)
		}

		return nil
	})
}

func (s *service) Cancel(ctx context.Context, subscriptionID models.NamespacedID, timing subscription.Timing) (subscription.Subscription, error) {
	ctx = subscription.NewSubscriptionOperationContext(ctx)

	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
		// First, let's get the subscription
		view, err := s.GetView(ctx, subscriptionID)
		if err != nil {
			return subscription.Subscription{}, err
		}

		if err := s.lockCustomer(ctx, view.Subscription.CustomerId); err != nil {
			return subscription.Subscription{}, err
		}

		if err := s.validateCancel(ctx, view, timing); err != nil {
			return subscription.Subscription{}, err
		}

		// Cancellation means that we deactivate everything by that deadline (set ActiveTo)
		// The different Cadences of the Spec are derived from the Subscription Cadence
		spec := view.AsSpec()

		cancelTime, err := timing.ResolveForSpec(view.Spec)
		if err != nil {
			return subscription.Subscription{}, fmt.Errorf("failed to get cancelation time: %w", err)
		}

		spec.ActiveTo = lo.ToPtr(cancelTime)

		// We can use sync to do this
		sub, err := s.sync(ctx, view, spec)
		if err != nil {
			return sub, err
		}

		// Let's fetch the view for event generation
		view, err = s.GetView(ctx, sub.NamespacedID)
		if err != nil {
			return sub, err
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		err = errors.Join(lo.Map(s.Hooks, func(v subscription.SubscriptionCommandHook, _ int) error {
			return v.AfterCancel(ctx, view)
		})...)
		if err != nil {
			return sub, fmt.Errorf("failed to validate subscription: %w", err)
		}

		err = s.Publisher.Publish(ctx, subscription.NewCancelledEvent(ctx, view))
		if err != nil {
			return sub, fmt.Errorf("failed to publish event: %w", err)
		}

		return sub, nil
	})
}

func (s *service) Continue(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	ctx = subscription.NewSubscriptionOperationContext(ctx)

	// First, let's get the subscription
	view, err := s.GetView(ctx, subscriptionID)
	if err != nil {
		return subscription.Subscription{}, err
	}

	if err := s.validateContinue(ctx, view); err != nil {
		return subscription.Subscription{}, err
	}

	// Continuation means, that we recalculate the deactivation deadlines as if there was no cancellation
	// This is handled by the SubscriptionSpec as all Cadences are derived from the Subscription Cadence
	spec := view.AsSpec()

	spec.ActiveTo = nil

	// We can use sync to do this
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		if err := errors.Join(lo.Map(s.Hooks, func(v subscription.SubscriptionCommandHook, _ int) error {
			return v.BeforeContinue(ctx, view)
		})...); err != nil {
			return subscription.Subscription{}, fmt.Errorf("failed to validate subscription: %w", err)
		}

		sub, err := s.sync(ctx, view, spec)
		if err != nil {
			return sub, err
		}

		// Let's fetch the view for event generation
		view, err := s.GetView(ctx, sub.NamespacedID)
		if err != nil {
			return sub, err
		}

		err = errors.Join(lo.Map(s.Hooks, func(v subscription.SubscriptionCommandHook, _ int) error {
			return v.AfterContinue(ctx, view)
		})...)
		if err != nil {
			return sub, fmt.Errorf("failed to validate subscription: %w", err)
		}

		err = s.Publisher.Publish(ctx, subscription.NewContinuedEvent(ctx, view))
		if err != nil {
			return sub, fmt.Errorf("failed to publish event: %w", err)
		}

		return sub, nil
	})
}

func (s *service) UpdateAnnotations(ctx context.Context, subscriptionID models.NamespacedID, annotations models.Annotations) (*subscription.Subscription, error) {
	return s.SubscriptionRepo.UpdateAnnotations(ctx, subscriptionID, annotations)
}

func (s *service) Get(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	ctx = subscription.NewSubscriptionOperationContext(ctx)

	sub, err := s.SubscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return subscription.Subscription{}, err
	}
	return sub, nil
}

func (s *service) GetView(ctx context.Context, subscriptionID models.NamespacedID) (subscription.SubscriptionView, error) {
	ctx = subscription.NewSubscriptionOperationContext(ctx)

	var def subscription.SubscriptionView

	sub, err := s.Get(ctx, subscriptionID)
	if err != nil {
		return def, err
	}

	cus, err := s.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: sub.Namespace,
			ID:        sub.CustomerId,
		},
	})
	if err != nil {
		return def, err
	}

	if cus == nil {
		return def, fmt.Errorf("customer is nil")
	}

	views, err := s.ExpandViews(ctx, []subscription.Subscription{sub})
	if err != nil {
		return def, fmt.Errorf("failed to get views: %w", err)
	}

	if len(views) != 1 {
		return def, fmt.Errorf("expected 1 view, got %d", len(views))
	}

	return views[0], nil
}

func (s *service) List(ctx context.Context, input subscription.ListSubscriptionsInput) (subscription.SubscriptionList, error) {
	ctx = subscription.NewSubscriptionOperationContext(ctx)

	if err := input.Validate(); err != nil {
		return subscription.SubscriptionList{}, fmt.Errorf("input is invalid: %w", err)
	}

	return s.SubscriptionRepo.List(ctx, input)
}

func (s *service) ExpandViews(ctx context.Context, subs []subscription.Subscription) ([]subscription.SubscriptionView, error) {
	ctx = subscription.NewSubscriptionOperationContext(ctx)

	if len(subs) == 0 {
		return nil, nil
	}

	// If we have multiple customer ids, we can't expand the views
	if len(lo.Uniq(slicesx.Map(subs, func(s subscription.Subscription) string {
		return s.CustomerId
	}))) != 1 {
		return nil, fmt.Errorf("ExpandViews only supports a single customer id for now")
	}

	customerID := subs[0].CustomerId

	cus, err := s.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: subs[0].Namespace,
			ID:        customerID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	if cus == nil {
		return nil, fmt.Errorf("customer is nil")
	}

	now := clock.Now()

	getAtInputs := slicesx.Map(subs, func(s subscription.Subscription) subscription.GetForSubscriptionAtInput {
		return subscription.GetForSubscriptionAtInput{
			Namespace:      s.Namespace,
			SubscriptionID: s.ID,
			At:             now,
		}
	})

	phases, err := s.SubscriptionPhaseRepo.GetForSubscriptionsAt(ctx, getAtInputs)
	if err != nil {
		return nil, fmt.Errorf("failed to get phases: %w", err)
	}

	items, err := s.SubscriptionItemRepo.GetForSubscriptionsAt(ctx, getAtInputs)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}

	ents, err := s.EntitlementAdapter.GetForSubscriptionsAt(ctx, getAtInputs)
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlements: %w", err)
	}

	var featsOfEnts pagination.Result[feature.Feature]

	{
		uniqFeatureIDs := lo.Uniq(slicesx.Map(ents, func(e subscription.SubscriptionEntitlement) string {
			return e.Entitlement.FeatureID
		}))

		if len(uniqFeatureIDs) > 0 {
			featsOfEnts, err = s.FeatureService.ListFeatures(ctx, feature.ListFeaturesParams{
				Namespace:       cus.Namespace,
				IncludeArchived: true,
				IDsOrKeys:       uniqFeatureIDs,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get features of entitlements: %w", err)
			}
		}
	}

	var featsOfItems pagination.Result[feature.Feature]

	{
		itemsWithFeatures := lo.Filter(items, func(i subscription.SubscriptionItem, _ int) bool {
			return i.RateCard.AsMeta().FeatureKey != nil
		})

		uniqFeatureKeys := lo.Uniq(slicesx.Map(itemsWithFeatures, func(i subscription.SubscriptionItem) string {
			return lo.FromPtr(i.RateCard.AsMeta().FeatureKey)
		}))

		if len(uniqFeatureKeys) > 0 {
			featsOfItems, err = s.FeatureService.ListFeatures(ctx, feature.ListFeaturesParams{
				Namespace:       cus.Namespace,
				IncludeArchived: true,
				IDsOrKeys:       uniqFeatureKeys,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get features of items: %w", err)
			}
		}
	}

	phasesBySub := lo.GroupBy(phases, func(p subscription.SubscriptionPhase) string {
		return p.SubscriptionID
	})

	if diff := numNotGrouped(phases, phasesBySub); diff > 0 {
		return nil, fmt.Errorf("%d phases are not grouped by subscription id", diff)
	}

	itemsBySub := lo.GroupBy(items, func(i subscription.SubscriptionItem) string {
		phase, ok := lo.Find(phases, func(p subscription.SubscriptionPhase) bool {
			return p.ID == i.PhaseId
		})
		if !ok {
			return ""
		}

		return phase.SubscriptionID
	})

	if diff := numNotGrouped(items, itemsBySub); diff > 0 {
		return nil, fmt.Errorf("%d items are not grouped by subscription id", diff)
	}

	entsBySub := lo.MapEntries(itemsBySub, func(key string, items []subscription.SubscriptionItem) (string, []subscription.SubscriptionEntitlement) {
		found := make([]subscription.SubscriptionEntitlement, 0)
		for _, item := range items {
			if item.EntitlementID == nil {
				continue
			}

			ent, ok := lo.Find(ents, func(e subscription.SubscriptionEntitlement) bool {
				return e.Entitlement.ID == *item.EntitlementID
			})
			if ok {
				found = append(found, ent)
			}
		}

		return key, found
	})

	if diff := numNotGrouped(ents, entsBySub); diff > 0 {
		return nil, fmt.Errorf("%d entitlements are not grouped by subscription id", diff)
	}

	featsOfEntsBySub := lo.MapEntries(entsBySub, func(key string, ents []subscription.SubscriptionEntitlement) (string, []feature.Feature) {
		found := make([]feature.Feature, 0)
		for _, ent := range ents {
			feat, ok := lo.Find(featsOfEnts.Items, func(f feature.Feature) bool {
				return f.ID == ent.Entitlement.FeatureID
			})
			if ok {
				found = append(found, feat)
			}
		}

		return key, found
	})

	if diff := numNotGrouped(featsOfEnts.Items, featsOfEntsBySub); diff > 0 {
		return nil, fmt.Errorf("%d features of entitlements are not grouped by subscription id", diff)
	}

	featsOfItemsBySub := lo.MapEntries(itemsBySub, func(key string, items []subscription.SubscriptionItem) (string, []feature.Feature) {
		found := make([]feature.Feature, 0)
		for _, item := range items {
			if item.RateCard.AsMeta().FeatureKey == nil {
				continue
			}

			feat, ok := lo.Find(featsOfItems.Items, func(f feature.Feature) bool {
				return f.Key == lo.FromPtr(item.RateCard.AsMeta().FeatureKey)
			})
			if ok {
				found = append(found, feat)
			}
		}

		return key, found
	})

	return slicesx.MapWithErr(subs, func(s subscription.Subscription) (subscription.SubscriptionView, error) {
		view, err := subscription.NewSubscriptionView(s, lo.FromPtr(cus),
			phasesBySub[s.ID],
			itemsBySub[s.ID],
			entsBySub[s.ID],
			featsOfEntsBySub[s.ID],
			featsOfItemsBySub[s.ID],
		)

		return lo.FromPtr(view), err
	})
}

func (s *service) updateCustomerCurrencyIfNotSet(ctx context.Context, sub subscription.Subscription, currentSpec subscription.SubscriptionSpec) error {
	cus, err := s.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: sub.Namespace,
			ID:        sub.CustomerId,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	if cus != nil && cus.IsDeleted() {
		return models.NewGenericPreConditionFailedError(
			fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
		)
	}

	if cus == nil {
		return fmt.Errorf("customer is nil")
	}

	// Let's set the customer's currency to the subscription currency for paid subscriptions (if not already set)
	if cus.Currency == nil && currentSpec.HasBillables() {
		if _, err := s.CustomerService.UpdateCustomer(ctx, customer.UpdateCustomerInput{
			CustomerID: cus.GetID(),
			CustomerMutate: customer.CustomerMutate{
				Name:             cus.Name,
				Key:              cus.Key,
				Description:      cus.Description,
				UsageAttribution: cus.UsageAttribution,
				PrimaryEmail:     cus.PrimaryEmail,
				BillingAddress:   cus.BillingAddress,
				Currency:         &currentSpec.Currency,
				Metadata:         cus.Metadata,
				Annotation:       cus.Annotation,
			},
		}); err != nil {
			return fmt.Errorf("failed to update customer currency: %w", err)
		}
	}

	return nil
}
