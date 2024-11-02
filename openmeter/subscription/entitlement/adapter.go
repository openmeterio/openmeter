package subscriptionentitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/annotations"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type EntitlementSubscriptionAdapter struct {
	entitlementConnector entitlement.Connector
	repo                 Repository
	txCreator            transaction.Creator
}

var _ subscription.EntitlementAdapter = &EntitlementSubscriptionAdapter{}

func NewEntitlementSubscriptionAdapter(
	entitlementConnector entitlement.Connector,
	repo Repository,
	txCreator transaction.Creator,
) *EntitlementSubscriptionAdapter {
	return &EntitlementSubscriptionAdapter{
		entitlementConnector: entitlementConnector,
		repo:                 repo,
		txCreator:            txCreator,
	}
}

func (a *EntitlementSubscriptionAdapter) ScheduleEntitlement(ctx context.Context, ref subscription.SubscriptionItemRef, input entitlement.CreateEntitlementInputs) (*subscription.SubscriptionEntitlement, error) {
	if input.ActiveFrom == nil {
		return nil, fmt.Errorf("active from is required")
	}

	at := *input.ActiveFrom

	ent, err := a.GetForItem(ctx, input.Namespace, ref, at)
	if err == nil {
		return nil, &AlreadyExistsError{
			ItemRef:       ref,
			EntitlementId: ent.Entitlement.ID,
		}
	} else if _, ok := lo.ErrorsAs[*NotFoundError](err); !ok {
		return nil, err
	}

	return transaction.Run(ctx, a.txCreator, func(ctx context.Context) (*subscription.SubscriptionEntitlement, error) {
		annotations.Annotate(input.Metadata, entitlement.SystemManagedEntitlementAnnotation)
		ent, err := a.entitlementConnector.ScheduleEntitlement(ctx, input)
		if err != nil {
			return nil, err
		}

		if ent == nil {
			return nil, fmt.Errorf("entitlement is nil")
		}

		sEnt, err := a.repo.Create(ctx, CreateSubscriptionEntitlementInput{
			Namespace:           ent.Namespace,
			EntitlementId:       ent.ID,
			SubscriptionItemRef: ref,
		})
		if err != nil {
			return nil, err
		}

		if sEnt == nil {
			return nil, fmt.Errorf("subscription entitlement is nil")
		}

		return &subscription.SubscriptionEntitlement{
			Entitlement: *ent,
			ItemRef:     sEnt.SubscriptionItemRef,
			Cadence: models.CadencedModel{
				ActiveFrom: *ent.ActiveFrom,
				ActiveTo:   ent.ActiveTo,
			},
		}, nil
	})
}

func (a *EntitlementSubscriptionAdapter) GetForItem(ctx context.Context, namespace string, ref subscription.SubscriptionItemRef, at time.Time) (*subscription.SubscriptionEntitlement, error) {
	sE, err := a.repo.GetBySubscriptionItem(ctx, namespace, ref, at)
	if err != nil {
		return nil, err
	}

	ent, err := a.entitlementConnector.GetEntitlement(ctx, sE.Namespace, sE.EntitlementId)
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
		ItemRef:     sE.SubscriptionItemRef,
		Cadence: models.CadencedModel{
			ActiveFrom: *ent.ActiveFrom,
			ActiveTo:   ent.ActiveTo,
		},
	}, nil
}

func (a *EntitlementSubscriptionAdapter) GetForSubscription(ctx context.Context, subscriptionID models.NamespacedID, at time.Time) ([]subscription.SubscriptionEntitlement, error) {
	sEnts, err := a.repo.GetForSubscription(ctx, subscriptionID, at)
	if err != nil {
		return nil, err
	}

	var ents pagination.PagedResponse[entitlement.Entitlement]

	if len(sEnts) > 0 {
		ents, err = a.entitlementConnector.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
			IDs:        lo.Map(sEnts, func(s SubscriptionEntitlement, _ int) string { return s.EntitlementId }),
			Namespaces: []string{subscriptionID.Namespace},
			Page:       pagination.Page{}, // zero value so all entitlements are fetched
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get Entitlements of SubscriptionEntitlements: %w", err)
		}
	}

	if len(ents.Items) != len(sEnts) {
		return nil, fmt.Errorf("entitlement count mismatch, expected %d, got %d", len(sEnts), len(ents.Items))
	}

	subEnts := make([]subscription.SubscriptionEntitlement, 0, len(sEnts))
	for i, sEnt := range sEnts {
		ent := ents.Items[i]

		if ent.ActiveFrom == nil {
			return nil, fmt.Errorf("entitlement active from is nil, entitlement doesn't have cadence")
		}

		subEnts = append(subEnts, subscription.SubscriptionEntitlement{
			Entitlement: ent,
			ItemRef:     sEnt.SubscriptionItemRef,
			Cadence: models.CadencedModel{
				ActiveFrom: *ent.ActiveFrom,
				ActiveTo:   ent.ActiveTo,
			},
		})
	}

	return subEnts, nil
}

func (a *EntitlementSubscriptionAdapter) Delete(ctx context.Context, namespace string, ref subscription.SubscriptionItemRef) error {
	sEnt, err := a.repo.GetBySubscriptionItem(ctx, namespace, ref, time.Now())
	if err != nil {
		return err
	}

	// Lets delete the entitlement first
	err = a.entitlementConnector.DeleteEntitlement(ctx, sEnt.Namespace, sEnt.EntitlementId, clock.Now())
	if err != nil {
		return fmt.Errorf("failed to delete Entitlement of SubscriptionEntitlement: %w", err)
	}
	// Then lets delete the subscription entitlement
	return a.repo.Delete(ctx, sEnt.ID)
}
