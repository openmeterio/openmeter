package repo

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
)

func MapDBSubscription(sub *db.Subscription) (subscription.Subscription, error) {
	if sub == nil {
		return subscription.Subscription{}, fmt.Errorf("unexpected nil subscription")
	}

	var ref *subscription.PlanRef

	if sub.Edges.Plan != nil {
		ref = &subscription.PlanRef{
			Id:      sub.Edges.Plan.ID,
			Key:     sub.Edges.Plan.Key,
			Version: sub.Edges.Plan.Version,
		}
		ref.Id = sub.Edges.Plan.ID
	}

	billingCadence, err := sub.BillingCadence.Parse()
	if err != nil {
		return subscription.Subscription{}, fmt.Errorf("failed to parse billing cadence: %w", err)
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
		MetadataModel: models.MetadataModel{
			Metadata: sub.Metadata,
		},
		Alignment: productcatalog.Alignment{
			BillablesMustAlign: sub.BillablesMustAlign,
		},
		PlanRef:         ref,
		Name:            sub.Name,
		Description:     sub.Description,
		CustomerId:      sub.CustomerID,
		Currency:        sub.Currency,
		BillingCadence:  billingCadence,
		ProRatingConfig: sub.ProRatingConfig,
		BillingAnchor:   sub.BillingAnchor.UTC(),
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
		MetadataModel: models.MetadataModel{
			Metadata: phase.Metadata,
		},
		ActiveFrom:     phase.ActiveFrom.UTC(),
		SubscriptionID: phase.SubscriptionID,
		Key:            phase.Key,
		Name:           phase.Name,
		Description:    phase.Description,
		SortHint:       phase.SortHint,
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

	var rc productcatalog.RateCard
	rcMeta := productcatalog.RateCardMeta{
		Name:                item.Name,
		Description:         item.Description,
		FeatureKey:          item.FeatureKey,
		EntitlementTemplate: item.EntitlementTemplate,
		TaxConfig:           item.TaxConfig,
		Price:               item.Price,
		Discounts:           lo.FromPtr(item.Discounts),
		Key:                 item.Key,
		FeatureID:           nil, // FIXME: is this an issue?
	}

	switch {
	case item.Price == nil:
		rc = &productcatalog.FlatFeeRateCard{
			BillingCadence: cadence,
			RateCardMeta:   rcMeta,
		}
	case item.Price.Type() == productcatalog.FlatPriceType:
		rc = &productcatalog.FlatFeeRateCard{
			BillingCadence: cadence,
			RateCardMeta:   rcMeta,
		}
	default:
		if cadence == nil {
			return subscription.SubscriptionItem{}, fmt.Errorf("billing cadence is required for usage based rate cards")
		}
		rc = &productcatalog.UsageBasedRateCard{
			BillingCadence: *cadence,
			RateCardMeta:   rcMeta,
		}
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
		MetadataModel: models.MetadataModel{
			Metadata: item.Metadata,
		},
		BillingBehaviorOverride: subscription.BillingBehaviorOverride{
			RestartBillingPeriod: item.RestartsBillingPeriod,
		},
		Annotations:                            item.Annotations,
		Name:                                   item.Name,
		Description:                            item.Description,
		ActiveFromOverrideRelativeToPhaseStart: sa,
		ActiveToOverrideRelativeToPhaseStart:   ea,
		SubscriptionId:                         phase.SubscriptionID,
		PhaseId:                                item.PhaseID,
		Key:                                    item.Key,
		EntitlementID:                          item.EntitlementID,
		RateCard:                               rc,
	}, nil
}
