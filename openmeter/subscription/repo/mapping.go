package repo

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
)

func MapDBSubscription(sub *db.Subscription) (subscription.Subscription, error) {
	if sub == nil {
		return subscription.Subscription{}, fmt.Errorf("unexpected nil subscription")
	}

	ref := subscription.PlanRef{
		Id: *sub.PlanID,
	}

	if sub.Edges.Plan != nil {
		ref.Key = sub.Edges.Plan.Key
		ref.Version = sub.Edges.Plan.Version
	}

	return subscription.Subscription{
		NamespacedID: models.NamespacedID{
			ID:        sub.ID,
			Namespace: sub.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: sub.CreatedAt.UTC(),
			UpdatedAt: sub.UpdatedAt.UTC(),
			DeletedAt: convert.SafeToUTC(sub.DeletedAt),
		},
		CadencedModel: models.CadencedModel{
			ActiveFrom: sub.ActiveFrom.UTC(),
			ActiveTo:   convert.SafeToUTC(sub.ActiveTo),
		},
		AnnotatedModel: models.AnnotatedModel{
			Metadata: sub.Metadata,
		},
		PlanRef:     ref,
		Name:        sub.Name,
		Description: sub.Description,
		CustomerId:  sub.CustomerID,
		Currency:    sub.Currency,
	}, nil
}

func MapDBSubscripitonPhase(phase *db.SubscriptionPhase) (subscription.SubscriptionPhase, error) {
	if phase == nil {
		return subscription.SubscriptionPhase{}, fmt.Errorf("unexpected nil subscription phase")
	}

	return subscription.SubscriptionPhase{
		NamespacedID: models.NamespacedID{
			ID:        phase.ID,
			Namespace: phase.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: phase.CreatedAt.UTC(),
			UpdatedAt: phase.UpdatedAt.UTC(),
			DeletedAt: convert.SafeToUTC(phase.DeletedAt),
		},
		AnnotatedModel: models.AnnotatedModel{
			Metadata: phase.Metadata,
		},
		ActiveFrom:     phase.ActiveFrom.UTC(),
		SubscriptionID: phase.SubscriptionID,
		Key:            phase.Key,
		Name:           phase.Name,
		Description:    phase.Description,
	}, nil
}

func MapDBSubscriptionItem(item *db.SubscriptionItem) (subscription.SubscriptionItem, error) {
	phase, err := item.Edges.PhaseOrErr()
	if err != nil {
		return subscription.SubscriptionItem{}, fmt.Errorf("failed to get phase for subscription item: %w", err)
	}

	if phase == nil {
		return subscription.SubscriptionItem{}, fmt.Errorf("unexpected nil phase for subscription item")
	}

	if item == nil {
		return subscription.SubscriptionItem{}, fmt.Errorf("unexpected nil subscription item")
	}

	sa, err := item.ActiveFromOverrideRelativeToPhaseStart.ParsePtrOrNil()
	if err != nil {
		return subscription.SubscriptionItem{}, fmt.Errorf("failed to parse start after phase: %w", err)
	}

	ea, err := item.ActiveToOverrideRelativeToPhaseStart.ParsePtrOrNil()
	if err != nil {
		return subscription.SubscriptionItem{}, fmt.Errorf("failed to parse end after phase: %w", err)
	}

	cadence, err := item.BillingCadence.ParsePtrOrNil()
	if err != nil {
		return subscription.SubscriptionItem{}, fmt.Errorf("failed to parse billing cadence: %w", err)
	}

	return subscription.SubscriptionItem{
		NamespacedID: models.NamespacedID{
			ID:        item.ID,
			Namespace: item.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: item.CreatedAt.UTC(),
			UpdatedAt: item.UpdatedAt.UTC(),
			DeletedAt: convert.SafeToUTC(item.DeletedAt),
		},
		CadencedModel: models.CadencedModel{
			ActiveFrom: item.ActiveFrom.UTC(),
			ActiveTo:   convert.SafeToUTC(item.ActiveTo),
		},
		AnnotatedModel: models.AnnotatedModel{
			Metadata: item.Metadata,
		},
		Name:                                   item.Name,
		Description:                            item.Description,
		ActiveFromOverrideRelativeToPhaseStart: sa,
		ActiveToOverrideRelativeToPhaseStart:   ea,
		SubscriptionId:                         phase.SubscriptionID,
		PhaseId:                                item.PhaseID,
		Key:                                    item.Key,
		EntitlementID:                          item.EntitlementID,
		RateCard: subscription.RateCard{
			Name:                item.Name,
			Description:         item.Description,
			FeatureKey:          item.FeatureKey,
			EntitlementTemplate: item.EntitlementTemplate,
			TaxConfig:           item.TaxConfig,
			Price:               item.Price,
			BillingCadence:      cadence,
		},
	}, nil
}
