package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ServiceConfig struct {
	SubscriptionRepo      subscription.SubscriptionRepository
	SubscriptionPhaseRepo subscription.SubscriptionPhaseRepository
	SubscriptionItemRepo  subscription.SubscriptionItemRepository
	// connectors
	CustomerService customer.Service
	// adapters
	EntitlementAdapter subscription.EntitlementAdapter
	// framework
	TransactionManager transaction.Creator
	Publisher          eventbus.Publisher
}

func New(conf ServiceConfig) subscription.Service {
	return &service{
		ServiceConfig: conf,
	}
}

type service struct {
	ServiceConfig
}

var _ subscription.Service = &service{}

func (s *service) Create(ctx context.Context, namespace string, spec subscription.SubscriptionSpec) (subscription.Subscription, error) {
	def := subscription.Subscription{}

	// Let's make sure Create is possible based on the transition rules
	if err := subscription.NewStateMachine(
		def.GetStatusAt(clock.Now()),
	).CanTransitionOrErr(ctx, subscription.SubscriptionActionCreate); err != nil {
		return def, err
	}

	// Valiate the spec
	if err := spec.Validate(); err != nil {
		return def, fmt.Errorf("spec is invalid: %w", err)
	}

	// Fetch the customer
	cust, err := s.CustomerService.GetCustomer(ctx, customerentity.GetCustomerInput{
		Namespace: namespace,
		ID:        spec.CustomerId,
	})
	if err != nil {
		return def, err
	}

	if cust == nil {
		return def, fmt.Errorf("customer is nil")
	}

	// Subscriptions can be canceled, scheduled to the future, etc...
	// To avoid difficult to resolve scheduling issues (think of overlapping, canceling, unscheduling cancels, etc...) we enforce a simple limit:
	// at a time only a single subscription can be scheduled, where scheduled means its either already active, or scheduled to be in the future.
	scheduled, err := s.SubscriptionRepo.GetAllForCustomerSince(ctx, models.NamespacedID{
		ID:        spec.CustomerId,
		Namespace: namespace,
	}, clock.Now())
	if err != nil {
		return def, fmt.Errorf("failed to get scheduled subscriptions: %w", err)
	}
	if len(scheduled) > 0 {
		return def, &models.GenericConflictError{Message: "customer already has subscriptions scheduled"}
	}

	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
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

		// Let's fetch the view for event generation
		view, err := s.GetView(ctx, sub.NamespacedID)
		if err != nil {
			return sub, err
		}

		err = s.Publisher.Publish(ctx, subscription.CreatedEvent{
			SubscriptionView: view,
		})
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

	// Let's make sure edit is possible based on the transition rules
	if err := subscription.NewStateMachine(
		view.Subscription.GetStatusAt(clock.Now()),
	).CanTransitionOrErr(ctx, subscription.SubscriptionActionUpdate); err != nil {
		return def, err
	}

	return s.sync(ctx, view, newSpec)
}

func (s *service) Cancel(ctx context.Context, subscriptionID models.NamespacedID, at time.Time) (subscription.Subscription, error) {
	// First, let's get the subscription
	view, err := s.GetView(ctx, subscriptionID)
	if err != nil {
		return subscription.Subscription{}, err
	}

	// Let's make sure Cancel is possible based on the transition rules
	if err := subscription.NewStateMachine(
		view.Subscription.GetStatusAt(clock.Now()),
	).CanTransitionOrErr(ctx, subscription.SubscriptionActionCancel); err != nil {
		return subscription.Subscription{}, err
	}

	spec := view.AsSpec()

	// Cancellation means that we deactivate everything by that deadline (set ActiveTo)
	// The different Cadences of the Spec are derived from the Subscription Cadence
	spec.ActiveTo = lo.ToPtr(at)

	if err := spec.Validate(); err != nil {
		return subscription.Subscription{}, fmt.Errorf("spec is invalid after setting cancelation time: %w", err)
	}

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

		err = s.Publisher.Publish(ctx, subscription.CreatedEvent{
			SubscriptionView: view,
		})
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

	// Let's make sure Continue is possible based on the transition rules
	if err := subscription.NewStateMachine(
		view.Subscription.GetStatusAt(clock.Now()),
	).CanTransitionOrErr(ctx, subscription.SubscriptionActionContinue); err != nil {
		return subscription.Subscription{}, err
	}

	// Continuation means, that we recalculate the deactivation deadlines as if there was no cancellation
	// This is handled by the SubscriptionSpec as all Cadences are derived from the Subscription Cadence
	spec := view.AsSpec()

	spec.ActiveTo = nil

	if err := spec.Validate(); err != nil {
		return subscription.Subscription{}, fmt.Errorf("spec is invalid after unsetting cancelation time: %w", err)
	}

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

		err = s.Publisher.Publish(ctx, subscription.CreatedEvent{
			SubscriptionView: view,
		})
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

	cust, err := s.CustomerService.GetCustomer(ctx, customerentity.GetCustomerInput{
		Namespace: sub.Namespace,
		ID:        sub.CustomerId,
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

	view, err := subscription.NewSubscriptionView(
		sub,
		*cust,
		phases,
		items,
		ents,
	)
	if err != nil {
		return def, err
	}

	if view == nil {
		return def, fmt.Errorf("view is nil")
	}

	return *view, nil
}
