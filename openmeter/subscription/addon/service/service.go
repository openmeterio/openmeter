package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Config struct {
	// repos
	SubAddRepo    subscriptionaddon.SubscriptionAddonRepository
	SubAddQtyRepo subscriptionaddon.SubscriptionAddonQuantityRepository

	Publisher eventbus.Publisher
	TxManager transaction.Creator

	// external
	AddonService     addon.Service
	PlanAddonService planaddon.Service
	SubService       subscription.Service
	Logger           *slog.Logger
}

func (c Config) Validate() error {
	if c.SubAddRepo == nil {
		return errors.New("subscription add-on repository is required")
	}

	if c.SubAddQtyRepo == nil {
		return errors.New("subscription add-on quantity repository is required")
	}

	if c.AddonService == nil {
		return errors.New("addon service is required")
	}

	if c.PlanAddonService == nil {
		return errors.New("plan add-on service is required")
	}

	if c.Publisher == nil {
		return errors.New("publisher is required")
	}

	if c.SubService == nil {
		return errors.New("subscription service is required")
	}

	if c.TxManager == nil {
		return errors.New("tx manager is required")
	}

	return nil
}

type service struct {
	cfg Config
}

var _ subscriptionaddon.Service = &service{}

func NewService(
	cfg Config,
) (subscriptionaddon.Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &service{
		cfg: cfg,
	}, nil
}

// Create creates a new subscription add-on
func (s *service) Create(ctx context.Context, ns string, input subscriptionaddon.CreateSubscriptionAddonInput) (*subscriptionaddon.SubscriptionAddon, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(fmt.Errorf("invalid input: %w", err))
	}

	add, err := s.cfg.AddonService.GetAddon(ctx, addon.GetAddonInput{
		NamespacedID: models.NamespacedID{
			Namespace: ns,
			ID:        input.AddonID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get add-on: %w", err)
	}

	if add == nil {
		return nil, fmt.Errorf("inconsitency error: nil add-on received")
	}

	if add.InstanceType == productcatalog.AddonInstanceTypeSingle && input.InitialQuantity.Quantity != 1 {
		return nil, models.NewGenericValidationError(errors.New("invalid input: single instance add-on must have initial quantity of 1"))
	}

	sView, err := s.cfg.SubService.GetView(ctx, models.NamespacedID{
		Namespace: ns,
		ID:        input.SubscriptionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if sView.Subscription.PlanRef == nil {
		return nil, models.NewGenericValidationError(errors.New("cannot add addon to a custom subscription"))
	}

	compatibility, err := s.cfg.PlanAddonService.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: ns,
		},
		PlanIDOrKey:  sView.Subscription.PlanRef.Id,
		AddonIDOrKey: input.AddonID,
	})
	if err != nil {
		if models.IsGenericNotFoundError(err) {
			return nil, models.NewGenericValidationError(fmt.Errorf("addon %s@%d is not linked to the plan %s@%d", add.Key, add.Version, sView.Subscription.PlanRef.Key, sView.Subscription.PlanRef.Version))
		}

		return nil, fmt.Errorf("failed to get plan add-on: %w", err)
	}

	phaseAtAddonStart, ok := sView.Spec.GetCurrentPhaseAt(input.InitialQuantity.ActiveFrom)
	if !ok {
		return nil, models.NewGenericValidationError(fmt.Errorf("subscription doesn't have an active phase at %s", input.InitialQuantity.ActiveFrom))
	}

	for _, phase := range sView.Phases {
		if phase.SubscriptionPhase.Key == compatibility.FromPlanPhase {
			// We reached the compatible start time first
			break
		}

		if phase.SubscriptionPhase.Key == phaseAtAddonStart.PhaseKey {
			return nil, models.NewGenericValidationError(fmt.Errorf("addon %s@%d can be only added starting with phase %s, current phase is %s", add.Key, add.Version, compatibility.FromPlanPhase, phaseAtAddonStart.PhaseKey))
		}
	}

	if compatibility.MaxQuantity != nil && input.InitialQuantity.Quantity > *compatibility.MaxQuantity {
		return nil, models.NewGenericValidationError(fmt.Errorf("addon %s@%d can be added a maximum of %d times", add.Key, add.Version, *compatibility.MaxQuantity))
	}

	return transaction.Run(ctx, s.cfg.TxManager, func(ctx context.Context) (*subscriptionaddon.SubscriptionAddon, error) {
		// Create the subscription addon
		subAdd, err := s.cfg.SubAddRepo.Create(ctx, ns, subscriptionaddon.CreateSubscriptionAddonRepositoryInput{
			MetadataModel:  input.MetadataModel,
			AddonID:        input.AddonID,
			SubscriptionID: input.SubscriptionID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create subscription add-on: %w", err)
		}

		if subAdd == nil {
			return nil, fmt.Errorf("inconsitency error: nil subscription add-on received")
		}

		// Create the initial quantity
		_, err = s.cfg.SubAddQtyRepo.Create(ctx, *subAdd, subscriptionaddon.CreateSubscriptionAddonQuantityRepositoryInput{
			ActiveFrom: input.InitialQuantity.ActiveFrom,
			Quantity:   input.InitialQuantity.Quantity,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create subscription add-on quantity: %w", err)
		}

		// Let's fetch the addon again and return it
		subscriptionAddon, err := s.cfg.SubAddRepo.Get(ctx, *subAdd)
		if err != nil {
			return nil, fmt.Errorf("failed to get subscription add-on: %w", err)
		}

		if subscriptionAddon == nil {
			return nil, fmt.Errorf("inconsitency error: nil subscription add-on received")
		}

		// Publish the event
		err = s.cfg.Publisher.Publish(ctx, subscriptionaddon.NewCreatedEvent(ctx, sView.Customer, *subscriptionAddon))
		if err != nil {
			return subscriptionAddon, fmt.Errorf("failed to publish event: %w", err)
		}

		return subscriptionAddon, nil
	})
}

// Get gets a subscription add-on
func (s *service) Get(ctx context.Context, id models.NamespacedID) (*subscriptionaddon.SubscriptionAddon, error) {
	return s.cfg.SubAddRepo.Get(ctx, id)
}

// List lists subscription add-ons
func (s *service) List(ctx context.Context, ns string, input subscriptionaddon.ListSubscriptionAddonsInput) (pagination.Result[subscriptionaddon.SubscriptionAddon], error) {
	def := pagination.Result[subscriptionaddon.SubscriptionAddon]{}
	if err := input.Validate(); err != nil {
		return def, fmt.Errorf("invalid input: %w", err)
	}

	return s.cfg.SubAddRepo.List(ctx, ns, subscriptionaddon.ListSubscriptionAddonRepositoryInput(input))
}

// ChangeQuantity changes the quantity of a subscription add-on
func (s *service) ChangeQuantity(ctx context.Context, id models.NamespacedID, input subscriptionaddon.CreateSubscriptionAddonQuantityInput) (*subscriptionaddon.SubscriptionAddon, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	subAdd, err := s.cfg.SubAddRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription add-on: %w", err)
	}

	if subAdd == nil {
		return nil, fmt.Errorf("subscription add-on not found")
	}

	add, err := s.cfg.AddonService.GetAddon(ctx, addon.GetAddonInput{
		NamespacedID: models.NamespacedID{
			Namespace: subAdd.Namespace,
			ID:        subAdd.Addon.ID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get add-on: %w", err)
	}

	if add == nil {
		return nil, fmt.Errorf("inconsitency error: nil add-on received")
	}

	if add.InstanceType == productcatalog.AddonInstanceTypeSingle && input.Quantity > 1 {
		return nil, models.NewGenericValidationError(errors.New("invalid input: single instance addon must have quantity of 1"))
	}

	sView, err := s.cfg.SubService.GetView(ctx, models.NamespacedID{
		Namespace: subAdd.Namespace,
		ID:        subAdd.SubscriptionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	compatibility, err := s.cfg.PlanAddonService.GetPlanAddon(ctx, planaddon.GetPlanAddonInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: subAdd.Namespace,
		},
		PlanIDOrKey:  sView.Subscription.PlanRef.Id,
		AddonIDOrKey: subAdd.Addon.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get plan add-on: %w", err)
	}

	phaseAtAddonStart, ok := sView.Spec.GetCurrentPhaseAt(input.ActiveFrom)
	if !ok {
		return nil, models.NewGenericValidationError(fmt.Errorf("subscription doesn't have an active phase at %s", input.ActiveFrom))
	}

	for _, phase := range sView.Phases {
		if phase.SubscriptionPhase.Key == compatibility.FromPlanPhase {
			// We reached the compatible start time first
			break
		}

		if phase.SubscriptionPhase.Key == phaseAtAddonStart.PhaseKey {
			return nil, models.NewGenericValidationError(fmt.Errorf("addon %s@%d can be only added starting with phase %s, current phase is %s", add.Key, add.Version, compatibility.FromPlanPhase, phaseAtAddonStart.PhaseKey))
		}
	}

	if compatibility.MaxQuantity != nil && input.Quantity > *compatibility.MaxQuantity {
		return nil, models.NewGenericValidationError(fmt.Errorf("addon %s@%d can be added a maximum of %d times", add.Key, add.Version, *compatibility.MaxQuantity))
	}

	return transaction.Run(ctx, s.cfg.TxManager, func(ctx context.Context) (*subscriptionaddon.SubscriptionAddon, error) {
		// Let's save the new quantity, there's no validation necessary
		_, err := s.cfg.SubAddQtyRepo.Create(ctx, id, subscriptionaddon.CreateSubscriptionAddonQuantityRepositoryInput(input))
		if err != nil {
			return nil, fmt.Errorf("failed to create subscription add-on quantity: %w", err)
		}

		// Let's fetch the addon and return it
		subscriptionAddon, err := s.cfg.SubAddRepo.Get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get subscription add-on: %w", err)
		}

		if subscriptionAddon == nil {
			return nil, fmt.Errorf("inconsitency error: nil subscription add-on received")
		}

		// Publish the event
		err = s.cfg.Publisher.Publish(ctx, subscriptionaddon.NewChangeQuantityEvent(ctx, sView.Customer, *subscriptionAddon))
		if err != nil {
			return subscriptionAddon, fmt.Errorf("failed to publish event: %w", err)
		}

		return subscriptionAddon, nil
	})
}
