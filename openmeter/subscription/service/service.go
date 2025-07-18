package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
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
	// External validations (optional)
	Validators []subscription.SubscriptionValidator
}

func New(conf ServiceConfig) subscription.Service {
	return &service{
		ServiceConfig: conf,
	}
}

var _ subscription.Service = &service{}

type service struct {
	ServiceConfig

	mu sync.RWMutex
}

func (s *service) RegisterValidator(validator subscription.SubscriptionValidator) error {
	if validator == nil {
		return errors.New("invalid subscription validator: nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.Validators = append(s.Validators, validator)

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
	def := subscription.Subscription{}

	// Fetch the customer & validate the customer
	cust, err := s.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: namespace,
			ID:        spec.CustomerId,
		},
	})
	if err != nil {
		return def, err
	}

	if cust == nil {
		return def, fmt.Errorf("customer is nil")
	}

	if err := s.validateCreate(ctx, *cust, spec); err != nil {
		return def, err
	}

	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
		if err := s.lockCustomer(ctx, spec.CustomerId); err != nil {
			return def, err
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

			if _, err := s.createPhase(ctx, *cust, *phase, sub, phaseCadence); err != nil {
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

		s.mu.RLock()
		defer s.mu.RUnlock()

		err = errors.Join(lo.Map(s.Validators, func(v subscription.SubscriptionValidator, _ int) error {
			return v.ValidateCreate(ctx, view)
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
	var def subscription.Subscription

	// Get the full view
	view, err := s.GetView(ctx, subscriptionID)
	if err != nil {
		return def, fmt.Errorf("failed to get view: %w", err)
	}

	if err := s.validateUpdate(ctx, view, newSpec); err != nil {
		return def, err
	}

	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
		subs, err := s.sync(ctx, view, newSpec)
		if err != nil {
			return subs, err
		}

		// Let's fetch the view for event generation
		updatedView, err := s.GetView(ctx, subs.NamespacedID)
		if err != nil {
			return subs, err
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		err = errors.Join(lo.Map(s.Validators, func(v subscription.SubscriptionValidator, _ int) error {
			return v.ValidateUpdate(ctx, view)
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

	err = errors.Join(lo.Map(s.Validators, func(v subscription.SubscriptionValidator, _ int) error {
		return v.ValidateDelete(ctx, view)
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

		err = errors.Join(lo.Map(s.Validators, func(v subscription.SubscriptionValidator, _ int) error {
			return v.ValidateCancel(ctx, view)
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
		sub, err := s.sync(ctx, view, spec)
		if err != nil {
			return sub, err
		}

		// Let's fetch the view for event generation
		view, err := s.GetView(ctx, sub.NamespacedID)
		if err != nil {
			return sub, err
		}

		s.mu.RLock()
		defer s.mu.RUnlock()

		err = errors.Join(lo.Map(s.Validators, func(v subscription.SubscriptionValidator, _ int) error {
			return v.ValidateContinue(ctx, view)
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

func (s *service) Get(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	sub, err := s.SubscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return subscription.Subscription{}, err
	}
	return sub, nil
}

func (s *service) GetView(ctx context.Context, subscriptionID models.NamespacedID) (subscription.SubscriptionView, error) {
	var def subscription.SubscriptionView
	currentTime := clock.Now()

	sub, err := s.Get(ctx, subscriptionID)
	if err != nil {
		return def, err
	}

	cust, err := s.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: sub.Namespace,
			ID:        sub.CustomerId,
		},
	})
	if err != nil {
		return def, err
	}
	if cust == nil {
		return def, fmt.Errorf("customer is nil")
	}

	phases, err := s.SubscriptionPhaseRepo.GetForSubscriptionAt(ctx, sub.NamespacedID, currentTime)
	if err != nil {
		return def, err
	}

	items, err := s.SubscriptionItemRepo.GetForSubscriptionAt(ctx, sub.NamespacedID, currentTime)
	if err != nil {
		return def, err
	}

	ents, err := s.EntitlementAdapter.GetForSubscriptionAt(ctx, sub.NamespacedID, currentTime)
	if err != nil {
		return def, err
	}

	entitlementFeatureIDs := lo.Map(ents, func(ent subscription.SubscriptionEntitlement, _ int) string {
		return ent.Entitlement.FeatureID
	})

	itemFeatureKeys := lo.Map(lo.Filter(items, func(i subscription.SubscriptionItem, _ int) bool {
		return i.RateCard.AsMeta().FeatureKey != nil
	}), func(item subscription.SubscriptionItem, _ int) string {
		return *item.RateCard.AsMeta().FeatureKey
	})

	featsOfEntsPaged, err := s.FeatureService.ListFeatures(ctx, feature.ListFeaturesParams{
		Namespace:       sub.Namespace,
		IncludeArchived: true, // We match by ID, so those exact features might since have been archived
		IDsOrKeys:       lo.Uniq(entitlementFeatureIDs),
	})
	if err != nil {
		return def, err
	}

	featsOfItemsPaged, err := s.FeatureService.ListFeatures(ctx, feature.ListFeaturesParams{
		Namespace:       sub.Namespace,
		IncludeArchived: false,
		IDsOrKeys:       lo.Uniq(itemFeatureKeys),
	})
	if err != nil {
		return def, err
	}

	view, err := subscription.NewSubscriptionView(
		sub,
		*cust,
		phases,
		items,
		ents,
		featsOfEntsPaged.Items,
		featsOfItemsPaged.Items,
	)
	if err != nil {
		return def, err
	}

	if view == nil {
		return def, fmt.Errorf("view is nil")
	}

	return *view, nil
}

func (s *service) GetAllForCustomerSince(ctx context.Context, customerID models.NamespacedID, at time.Time) ([]subscription.Subscription, error) {
	return s.SubscriptionRepo.GetAllForCustomerSince(ctx, customerID, at)
}

func (s *service) List(ctx context.Context, input subscription.ListSubscriptionsInput) (subscription.SubscriptionList, error) {
	if err := input.Validate(); err != nil {
		return subscription.SubscriptionList{}, fmt.Errorf("input is invalid: %w", err)
	}

	return s.SubscriptionRepo.List(ctx, input)
}

func (s *service) updateCustomerCurrencyIfNotSet(ctx context.Context, sub subscription.Subscription, currentSpec subscription.SubscriptionSpec) error {
	cust, err := s.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: sub.Namespace,
			ID:        sub.CustomerId,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	if cust == nil {
		return fmt.Errorf("customer is nil")
	}

	// Let's set the customer's currency to the subscription currency for paid subscriptions (if not already set)
	if cust.Currency == nil && currentSpec.HasBillables() {
		if _, err := s.CustomerService.UpdateCustomer(ctx, customer.UpdateCustomerInput{
			CustomerID: cust.GetID(),
			CustomerMutate: customer.CustomerMutate{
				Name:             cust.Name,
				Key:              cust.Key,
				Description:      cust.Description,
				UsageAttribution: cust.UsageAttribution,
				PrimaryEmail:     cust.PrimaryEmail,
				BillingAddress:   cust.BillingAddress,
				Currency:         &currentSpec.Currency,
				Metadata:         cust.Metadata,
				Annotation:       cust.Annotation,
			},
		}); err != nil {
			return fmt.Errorf("failed to update customer currency: %w", err)
		}
	}

	return nil
}
