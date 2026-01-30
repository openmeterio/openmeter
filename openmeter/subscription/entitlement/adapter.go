package subscriptionentitlement

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type EntitlementSubscriptionAdapter struct {
	entitlementConnector entitlement.Service
	itemRepo             subscription.SubscriptionItemRepository
	txCreator            transaction.Creator
}

var _ subscription.EntitlementAdapter = &EntitlementSubscriptionAdapter{}

func NewSubscriptionEntitlementAdapter(
	entitlementConnector entitlement.Service,
	itemRepo subscription.SubscriptionItemRepository,
	txCreator transaction.Creator,
) *EntitlementSubscriptionAdapter {
	return &EntitlementSubscriptionAdapter{
		entitlementConnector: entitlementConnector,
		itemRepo:             itemRepo,
		txCreator:            txCreator,
	}
}

// TODO: implement usageMigration as needed
func (a *EntitlementSubscriptionAdapter) ScheduleEntitlement(ctx context.Context, input subscription.ScheduleSubscriptionEntitlementInput, annotations models.Annotations) (*subscription.SubscriptionEntitlement, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	return transaction.Run(ctx, a.txCreator, func(ctx context.Context) (*subscription.SubscriptionEntitlement, error) {
		input.CreateEntitlementInputs.SubscriptionManaged = true

		// Initialize annotations if not already set
		if input.CreateEntitlementInputs.Annotations == nil {
			input.CreateEntitlementInputs.Annotations = models.Annotations{}
		}

		// Add subscription annotations
		for k, v := range annotations {
			input.CreateEntitlementInputs.Annotations[k] = v
		}

		ent, err := a.entitlementConnector.ScheduleEntitlement(ctx, input.CreateEntitlementInputs)
		if err != nil {
			return nil, err
		}

		if ent == nil {
			return nil, fmt.Errorf("entitlement is nil")
		}

		return &subscription.SubscriptionEntitlement{
			Entitlement: entitlement.EntitlementWithCustomer{
				Entitlement: *ent,
				Customer:    input.Customer,
			},
			Cadence: models.CadencedModel{
				ActiveFrom: *ent.ActiveFrom,
				ActiveTo:   ent.ActiveTo,
			},
		}, nil
	})
}

// TODO: is this in use?
func (a *EntitlementSubscriptionAdapter) GetByItemID(ctx context.Context, id models.NamespacedID) (*subscription.SubscriptionEntitlement, error) {
	item, err := a.itemRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if item.EntitlementID == nil {
		return nil, &NotFoundError{ItemID: id}
	}

	ent, err := a.entitlementConnector.GetEntitlementWithCustomer(ctx, item.Namespace, *item.EntitlementID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Entitlement of SubscriptionEntitlement: %w", err)
	}

	if ent == nil {
		return nil, fmt.Errorf("entitlement is nil")
	}

	if ent.ActiveFrom == nil {
		return nil, fmt.Errorf("entitlement active from is nil, entitlement doesn't have cadence")
	}

	return &subscription.SubscriptionEntitlement{
		Entitlement: *ent,
		Cadence: models.CadencedModel{
			ActiveFrom: *ent.ActiveFrom,
			ActiveTo:   ent.ActiveTo,
		},
	}, nil
}

// TODO: is this in use?
func (a *EntitlementSubscriptionAdapter) GetForSubscriptionAt(ctx context.Context, input subscription.GetForSubscriptionAtInput) ([]subscription.SubscriptionEntitlement, error) {
	items, err := a.itemRepo.GetForSubscriptionAt(ctx, input)
	if err != nil {
		return nil, err
	}

	var ents pagination.Result[entitlement.Entitlement]

	items = lo.Filter(items, func(s subscription.SubscriptionItem, _ int) bool { return s.EntitlementID != nil })

	var customer *customer.Customer
	if len(items) > 0 {
		entsInScope, err := a.entitlementConnector.ListEntitlementsWithCustomer(ctx, entitlement.ListEntitlementsParams{
			IDs:        lo.Map(items, func(s subscription.SubscriptionItem, _ int) string { return *s.EntitlementID }),
			Namespaces: []string{input.Namespace},
			Page:       pagination.Page{}, // zero value so all entitlements are fetched
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get Entitlements of SubscriptionEntitlements: %w", err)
		}

		ents = entsInScope.Entitlements

		customer = entsInScope.CustomersByID[models.NamespacedID{Namespace: input.Namespace, ID: entsInScope.Entitlements.Items[0].CustomerID}]
		if customer == nil {
			return nil, fmt.Errorf("customer not found for entitlement %s", entsInScope.Entitlements.Items[0].ID)
		}
	}

	if len(ents.Items) != len(items) {
		return nil, fmt.Errorf("entitlement count mismatch, expected %d, got %d", len(items), len(ents.Items))
	}

	subEnts := make([]subscription.SubscriptionEntitlement, 0, len(items))
	for _, ent := range ents.Items {
		if ent.ActiveFrom == nil {
			return nil, fmt.Errorf("entitlement active from is nil, entitlement doesn't have cadence")
		}

		subEnts = append(subEnts, subscription.SubscriptionEntitlement{
			Entitlement: entitlement.EntitlementWithCustomer{
				Entitlement: ent,
				Customer:    *customer,
			},
			Cadence: models.CadencedModel{
				ActiveFrom: *ent.ActiveFrom,
				ActiveTo:   ent.ActiveTo,
			},
		})
	}

	return subEnts, nil
}

func (a *EntitlementSubscriptionAdapter) GetForSubscriptionsAt(ctx context.Context, input []subscription.GetForSubscriptionAtInput) ([]subscription.SubscriptionEntitlement, error) {
	items, err := a.itemRepo.GetForSubscriptionsAt(ctx, input)
	if err != nil {
		return nil, err
	}

	items = lo.Filter(items, func(s subscription.SubscriptionItem, _ int) bool { return s.EntitlementID != nil })

	if len(items) == 0 {
		return nil, nil
	}

	ents, err := a.entitlementConnector.ListEntitlementsWithCustomer(ctx, entitlement.ListEntitlementsParams{
		IDs:        lo.Map(items, func(s subscription.SubscriptionItem, _ int) string { return *s.EntitlementID }),
		Namespaces: lo.Uniq(lo.Map(input, func(s subscription.GetForSubscriptionAtInput, _ int) string { return s.Namespace })),
		Page:       pagination.Page{}, // zero value so all entitlements are fetched
	})
	if err != nil {
		return nil, err
	}

	return slicesx.MapWithErr(ents.Entitlements.Items, func(ent entitlement.Entitlement) (subscription.SubscriptionEntitlement, error) {
		if ent.ActiveFrom == nil {
			return subscription.SubscriptionEntitlement{}, fmt.Errorf("entitlement active from is nil, entitlement doesn't have cadence")
		}

		cust, ok := ents.CustomersByID[models.NamespacedID{Namespace: ent.Namespace, ID: ent.CustomerID}]
		if !ok || cust == nil {
			return subscription.SubscriptionEntitlement{}, fmt.Errorf("customer not found for entitlement %s", ent.ID)
		}

		return subscription.SubscriptionEntitlement{
			Entitlement: entitlement.EntitlementWithCustomer{
				Entitlement: ent,
				Customer:    *cust,
			},
			Cadence: models.CadencedModel{
				ActiveFrom: *ent.ActiveFrom,
				ActiveTo:   ent.ActiveTo,
			},
		}, nil
	})
}

func (a *EntitlementSubscriptionAdapter) DeleteByItemID(ctx context.Context, id models.NamespacedID) error {
	item, err := a.itemRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if item.EntitlementID == nil {
		return &NotFoundError{ItemID: id}
	}

	// Let's delete the entitlement now
	return a.entitlementConnector.DeleteEntitlement(ctx, item.Namespace, *item.EntitlementID, clock.Now())
}
