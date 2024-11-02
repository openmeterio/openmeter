package subscription

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription/price"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Comparable[T any] interface {
	Self() T
	Equal(T) bool
}

type Specced[T any] interface {
	AsSpec() T
}

// EntityManager is responsible for keeping a certain entity's state in sync with a desired state.
type EntityManager[TSpecBase any, TSpec Comparable[TSpecBase], TView Specced[TSpec], TEntity any] interface {
	// SyncState transforms a given entity from the current view to the desired spec.
	SyncState(ctx context.Context, current TView, target TSpec) (*TEntity, error)
}

// TODO: specify types
type (
	EntitlementManager = EntityManager[SubscriptionEntitlementSpec, *SubscriptionEntitlementSpec, *SubscriptionEntitlement, SubscriptionEntitlement]
	PriceManager       = EntityManager[CreatePriceSpec, *CreatePriceSpec, *SubscriptionPrice, price.Price]
	// SubscriptionManager DOES NOT manage phases and items, only the subscription resource itself.
	// To manage phases and items, use connector.Sync!
	SubscriptionManager = EntityManager[SubscriptionSpec, *SubscriptionSpec, *subscriptionView, Subscription]
)

// Some local utils to dedupe code
type cud[TSpecBase any, TSpec Comparable[TSpecBase], TView Specced[TSpec], TEntity any] interface {
	Create(ctx context.Context, target TSpec) (*TEntity, error)
	Update(ctx context.Context, current TView, target TSpec) (*TEntity, error)
	Delete(ctx context.Context, current TView) error
}

type syncer[TSpecBase any, TSpec Comparable[TSpecBase], TView Specced[TSpec], TEntity any] struct {
	fn func(ctx context.Context, current TView, target TSpec) (*TEntity, error)
}

func (s *syncer[TSpecBase, TSpec, TView, TEntity]) SyncState(ctx context.Context, current TView, target TSpec) (*TEntity, error) {
	return s.fn(ctx, current, target)
}

var _ EntityManager[any, Comparable[any], Specced[Comparable[any]], any] = &syncer[any, Comparable[any], Specced[Comparable[any]], any]{}

func managerFromCud[TSpecBase any, TSpec Comparable[TSpecBase], TView Specced[TSpec], TEntity any](cudder cud[TSpecBase, TSpec, TView, TEntity]) EntityManager[TSpecBase, TSpec, TView, TEntity] {
	return &syncer[TSpecBase, TSpec, TView, TEntity]{
		fn: func(ctx context.Context, current TView, target TSpec) (*TEntity, error) {
			if lo.IsNil(current) && lo.IsNil(target) {
				// This is a no-op, nothing was there previously and nothing is desired.
				// Alternatively we could return an error here
				return nil, nil
			}

			if lo.IsNil(current) && !lo.IsNil(target) {
				return cudder.Create(ctx, target)
			}

			if !lo.IsNil(current) && lo.IsNil(target) {
				err := cudder.Delete(ctx, current)
				return nil, err
			}

			if !lo.IsNil(current) && !lo.IsNil(target) {
				if !current.AsSpec().Equal(target.Self()) {
					return cudder.Update(ctx, current, target)
				} else {
					// If they're the same we don't need to do anything
					return nil, nil
				}
			}

			return nil, fmt.Errorf("unexpected state: current=%v, target=%v, but neither can be nil", current, target)
		},
	}
}

type entitlementCud struct {
	entitlementAdapter EntitlementAdapter
}

var _ cud[SubscriptionEntitlementSpec, *SubscriptionEntitlementSpec, *SubscriptionEntitlement, SubscriptionEntitlement] = &entitlementCud{}

func NewEntitlementManager(entitlementAdapter EntitlementAdapter) EntitlementManager {
	return managerFromCud(&entitlementCud{
		entitlementAdapter: entitlementAdapter,
	})
}

func (c *entitlementCud) Create(ctx context.Context, target *SubscriptionEntitlementSpec) (*SubscriptionEntitlement, error) {
	// FIXME: ID wont be present on this?
	return c.entitlementAdapter.ScheduleEntitlement(ctx, target.ItemRef, target.EntitlementInputs)
}

func (c *entitlementCud) Update(ctx context.Context, current *SubscriptionEntitlement, target *SubscriptionEntitlementSpec) (*SubscriptionEntitlement, error) {
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

func (c *entitlementCud) Delete(ctx context.Context, current *SubscriptionEntitlement) error {
	return c.entitlementAdapter.Delete(ctx, current.Entitlement.Namespace, current.ItemRef)
}

type priceCud struct {
	connector      price.Connector
	billingAdapter BillingAdapter
}

var _ cud[CreatePriceSpec, *CreatePriceSpec, *SubscriptionPrice, price.Price] = &priceCud{}

func NewPriceManager(connector price.Connector, billingAdapter BillingAdapter) PriceManager {
	return managerFromCud(&priceCud{
		connector:      connector,
		billingAdapter: billingAdapter,
	})
}

func (c *priceCud) Create(ctx context.Context, target *CreatePriceSpec) (*price.Price, error) {
	// FIXME: ID wont be present on this?
	return c.connector.Create(ctx, target.CreateInput)
}

func (c *priceCud) Update(ctx context.Context, current *SubscriptionPrice, target *CreatePriceSpec) (*price.Price, error) {
	// Modification to a price CANNOT take place effective before the price was last invoiced
	itemLastInvoiced, err := c.billingAdapter.ItemLastInvoicedAt(ctx, current.Namespace, current.AsSpec().SubscriptionItemRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get last invoiced time while syncing an update: %w", err)
	}
	currentTime := clock.Now()

	// paranoia check
	if itemLastInvoiced != nil && !currentTime.After(*itemLastInvoiced) {
		return nil, fmt.Errorf("price cannot be updated before it was last invoiced")
	}

	// When "updating" a price, either
	// 1. the price configuration changes (value, key..., billingCadence)
	if target.CreateInput.Spec != current.AsSpec().CreateInput.Spec {
		// If the price has entered it's active period, we have to close it effective now and create a new price
		if !current.CadencedModel.ActiveFrom.After(currentTime) {
			_, err := c.connector.EndCadence(ctx, models.NamespacedID{
				ID:        current.ID,
				Namespace: current.Namespace,
			}, &currentTime)
			if err != nil {
				return nil, fmt.Errorf("failed to end cadence while syncing an update: %w", err)
			}

			return c.Create(ctx, &CreatePriceSpec{
				SubscriptionItemRef: target.SubscriptionItemRef,
				CreateInput:         target.CreateInput,
				Cadence: models.CadencedModel{
					ActiveFrom: currentTime,
					ActiveTo:   target.Cadence.ActiveTo,
				},
			})
		} else {
			// If not then we can just delete and recreate it (equivalent to an update)
			err := c.Delete(ctx, current)
			if err != nil {
				return nil, fmt.Errorf("failed to delete while syncing an update: %w", err)
			}
			return c.Create(ctx, target)
		}
		// 2. or the active period changes (either due to delaying or ending)
	} else if target.Cadence != current.CadencedModel {
		if target.Cadence.ActiveFrom != current.CadencedModel.ActiveFrom {
			// We have to delete and recreate the price
			err := c.Delete(ctx, current)
			if err != nil {
				return nil, fmt.Errorf("failed to delete while syncing an update: %w", err)
			}
			return c.Create(ctx, target)
		} else {
			// Otherwise we can just end the cadence
			return c.connector.EndCadence(ctx, models.NamespacedID{
				ID:        current.ID,
				Namespace: current.Namespace,
			}, target.Cadence.ActiveTo)
		}
	}

	panic("not implemented")
}

func (c *priceCud) Delete(ctx context.Context, current *SubscriptionPrice) error {
	// Modification to a price CANNOT take place effective before the price was last invoiced
	itemLastInvoiced, err := c.billingAdapter.ItemLastInvoicedAt(ctx, current.Namespace, current.AsSpec().SubscriptionItemRef)
	if err != nil {
		return fmt.Errorf("failed to get last invoiced time while syncing a delete: %w", err)
	}
	currentTime := clock.Now()

	// paranoia check
	if itemLastInvoiced != nil && !currentTime.After(*itemLastInvoiced) {
		return fmt.Errorf("price cannot be deleted before it was last invoiced")
	}
	// If the price has already entered it's active period, we have to close it effective now, otherwise we can delete it
	if !current.CadencedModel.ActiveFrom.After(currentTime) {
		_, err := c.connector.EndCadence(ctx, models.NamespacedID{
			ID:        current.ID,
			Namespace: current.Namespace,
		}, &currentTime)
		return err
	} else {
		return c.connector.Delete(ctx, models.NamespacedID{
			ID:        current.ID,
			Namespace: current.Namespace,
		})
	}
}

type subscriptionCud struct {
	repo Repository
}

var _ cud[SubscriptionSpec, *SubscriptionSpec, *subscriptionView, Subscription] = &subscriptionCud{}

func NewSubscriptionManager(repo Repository) SubscriptionManager {
	return managerFromCud(&subscriptionCud{repo: repo})
}

func (c *subscriptionCud) Create(ctx context.Context, target *SubscriptionSpec) (*Subscription, error) {
	sub, err := c.repo.CreateSubscription(ctx, target.Namespace, target.GetCreateInput())
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (c *subscriptionCud) Update(ctx context.Context, current *subscriptionView, target *SubscriptionSpec) (*Subscription, error) {
	// Properties we can update are:
	// - TODO: PlanRef
	// - activeTo
	if target.ActiveTo != current.subscription.ActiveTo {
		return c.repo.EndCadence(ctx, current.subscription.ID, target.ActiveTo)
	}
	panic("not implemented")
}

func (c *subscriptionCud) Delete(ctx context.Context, current *subscriptionView) error {
	return fmt.Errorf("entitymanager cannot delete a subscription!")
}
