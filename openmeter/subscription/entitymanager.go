package subscription

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription/price"
)

// EntityManager is responsible for keeping a certain entity's state in sync with a desired state.
type EntityManager[TView, TSpec, TEntity any] interface {
	// SyncState transforms a given entity from the current view to the desired spec.
	SyncState(ctx context.Context, current *TView, target *TSpec) (*TEntity, error)
}

// TODO: specify types
type (
	EntitlementManager = EntityManager[SubscriptionEntitlement, SubscriptionEntitlementSpec, SubscriptionEntitlement]
	PriceManager       = EntityManager[price.Price, CreatePriceSpec, price.Price]
	// SubscriptionManager DOES NOT manage phases and items, only the subscription resource itself.
	// To manage phases and items, use connector.Sync!
	SubscriptionManager = EntityManager[subscriptionView, SubscriptionSpec, Subscription]
)

// Some local utils to dedupe code
type cud[TView, TSpec, TEntity any] interface {
	Create(ctx context.Context, target TSpec) (*TEntity, error)
	Update(ctx context.Context, current TView, target TSpec) (*TEntity, error)
	Delete(ctx context.Context, current TView) error
}

type syncer[TView, TSpec, TEntity any] struct {
	fn func(ctx context.Context, current *TView, target *TSpec) (*TEntity, error)
}

func (s *syncer[TView, TSpec, TEntity]) SyncState(ctx context.Context, current *TView, target *TSpec) (*TEntity, error) {
	return s.fn(ctx, current, target)
}

func managerFromCud[TView, TSpec, TEntity any](cudder cud[TView, TSpec, TEntity]) EntityManager[TView, TSpec, TEntity] {
	return &syncer[TView, TSpec, TEntity]{
		fn: func(ctx context.Context, current *TView, target *TSpec) (*TEntity, error) {
			if lo.IsNil(current) && lo.IsNil(target) {
				// This is a no-op, nothing was there previously and nothing is desired.
				// Alternatively we could return an error here
				return nil, nil
			}

			if lo.IsNil(current) && !lo.IsNil(target) {
				return cudder.Create(ctx, *target)
			}

			if !lo.IsNil(current) && !lo.IsNil(target) {
				return cudder.Update(ctx, *current, *target)
			}

			if !lo.IsNil(current) && lo.IsNil(target) {
				err := cudder.Delete(ctx, *current)
				return nil, err
			}

			return nil, fmt.Errorf("unexpected state: current=%v, target=%v, but neither can be nil", current, target)
		},
	}
}

type entitlementCud struct {
	entitlementAdapter EntitlementAdapter
}

var _ cud[SubscriptionEntitlement, SubscriptionEntitlementSpec, SubscriptionEntitlement] = &entitlementCud{}

func NewEntitlementManager(entitlementAdapter EntitlementAdapter) EntitlementManager {
	return managerFromCud(&entitlementCud{
		entitlementAdapter: entitlementAdapter,
	})
}

func (c *entitlementCud) Create(ctx context.Context, target SubscriptionEntitlementSpec) (*SubscriptionEntitlement, error) {
	// FIXME: ID wont be present on this?
	return c.entitlementAdapter.ScheduleEntitlement(ctx, target.ItemRef, target.EntitlementInputs)
}

func (c *entitlementCud) Update(ctx context.Context, current SubscriptionEntitlement, target SubscriptionEntitlementSpec) (*SubscriptionEntitlement, error) {
	// How do we update entitlements?
	// Entitlements don't have update methods, the naive approach is that we just delete the current one and create a new one.
	// Are there any changes that have to be made?
	// For now, let's naively assume no!
	err := c.Delete(ctx, current)
	if err != nil {
		return nil, err
	}
	return c.Create(ctx, target)
}

func (c *entitlementCud) Delete(ctx context.Context, current SubscriptionEntitlement) error {
	return c.entitlementAdapter.Delete(ctx, current.Entitlement.Namespace, current.ItemRef)
}

type priceCud struct {
	connector price.Connector
}

var _ cud[price.Price, CreatePriceSpec, price.Price] = &priceCud{}

func NewPriceManager(connector price.Connector) PriceManager {
	return managerFromCud(&priceCud{
		connector: connector,
	})
}

func (c *priceCud) Create(ctx context.Context, target CreatePriceSpec) (*price.Price, error) {
	// FIXME: ID wont be present on this?
	return c.connector.Create(ctx, target.CreateInput)
}

func (c *priceCud) Update(ctx context.Context, current price.Price, target CreatePriceSpec) (*price.Price, error) {
	// FIXME: We cannot update prices if they were already invoiced
	// In that case we'd have to unschedul the price and create a new one from the current timestamp
	panic("not implemented")
}

func (c *priceCud) Delete(ctx context.Context, current price.Price) error {
	// FIXME: We cannot delete prices if they were already invoiced
	panic("not implemented")
}

type subscriptionCud struct {
	repo Repository
}

var _ cud[subscriptionView, SubscriptionSpec, Subscription] = &subscriptionCud{}

func NewSubscriptionManager(repo Repository) SubscriptionManager {
	return managerFromCud(&subscriptionCud{repo: repo})
}

func (c *subscriptionCud) Create(ctx context.Context, target SubscriptionSpec) (*Subscription, error) {
	sub, err := c.repo.CreateSubscription(ctx, target.Namespace, target.GetCreateInput())
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (c *subscriptionCud) Update(ctx context.Context, current subscriptionView, target SubscriptionSpec) (*Subscription, error) {
	// Properties we can update are:
	// - TODO: PlanRef
	// - activeTo
	if target.ActiveTo != current.subscription.ActiveTo && target.ActiveTo != nil {
		return c.repo.EndCadence(ctx, current.subscription.ID, *target.ActiveTo)
	}
	panic("not implemented")
}

func (c *subscriptionCud) Delete(ctx context.Context, current subscriptionView) error {
	return fmt.Errorf("entitymanager cannot delete a subscription!")
}
