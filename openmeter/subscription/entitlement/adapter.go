package subscriptionentitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/annotations"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

type EntitlementSubscriptionAdapter struct {
	entitlementConnector entitlement.Connector
	repo                 Repository
	txCreator            transaction.Creator
}

var _ subscription.EntitlementAdapter = &EntitlementSubscriptionAdapter{}

func NewEntitlementSubscriptionAdapter() *EntitlementSubscriptionAdapter {
	return &EntitlementSubscriptionAdapter{}
}

func (a *EntitlementSubscriptionAdapter) ScheduleEntitlement(ctx context.Context, ref subscription.SubscriptionItemRef, input entitlement.CreateEntitlementInputs) (*subscription.SubscriptionEntitlement, error) {
	if input.ActiveFrom == nil {
		return nil, fmt.Errorf("active from is required")
	}

	at := *input.ActiveFrom

	ent, err := a.GetForItem(ctx, ref, at)
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

func (a *EntitlementSubscriptionAdapter) GetForItem(ctx context.Context, ref subscription.SubscriptionItemRef, at time.Time) (*subscription.SubscriptionEntitlement, error) {
	sE, err := a.repo.GetBySubscriptionItem(ctx, ref, at)
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
	panic("implement me")
}
