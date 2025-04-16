package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Config struct {
	// repos
	SubAddRepo    subscriptionaddon.SubscriptionAddonRepository
	SubAddQtyRepo subscriptionaddon.SubscriptionAddonQuantityRepository

	TxManager transaction.Creator

	// external
	AddonService addon.Service
	SubService   subscription.Service
	Logger       *slog.Logger
}

type service struct {
	cfg Config
}

var _ subscriptionaddon.Service = &service{}

func NewService(
	cfg Config,
) subscriptionaddon.Service {
	return &service{
		cfg: cfg,
	}
}

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

	_, err = s.cfg.SubService.GetView(ctx, models.NamespacedID{
		Namespace: ns,
		ID:        input.SubscriptionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
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
		return s.cfg.SubAddRepo.Get(ctx, *subAdd)
	})
}

func (s *service) Get(ctx context.Context, id models.NamespacedID) (*subscriptionaddon.SubscriptionAddon, error) {
	return s.cfg.SubAddRepo.Get(ctx, id)
}

func (s *service) List(ctx context.Context, ns string, input subscriptionaddon.ListSubscriptionAddonsInput) (pagination.PagedResponse[subscriptionaddon.SubscriptionAddon], error) {
	def := pagination.PagedResponse[subscriptionaddon.SubscriptionAddon]{}
	if err := input.Validate(); err != nil {
		return def, fmt.Errorf("invalid input: %w", err)
	}

	return s.cfg.SubAddRepo.List(ctx, ns, subscriptionaddon.ListSubscriptionAddonRepositoryInput(input))
}

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

	return transaction.Run(ctx, s.cfg.TxManager, func(ctx context.Context) (*subscriptionaddon.SubscriptionAddon, error) {
		// Let's save the new quantity, there's no validation necessary
		_, err := s.cfg.SubAddQtyRepo.Create(ctx, id, subscriptionaddon.CreateSubscriptionAddonQuantityRepositoryInput(input))
		if err != nil {
			return nil, fmt.Errorf("failed to create subscription add-on quantity: %w", err)
		}

		// Let's fetch the addon and return it
		return s.cfg.SubAddRepo.Get(ctx, id)
	})
}
