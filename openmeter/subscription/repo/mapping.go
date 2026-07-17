package repo

import (
	"fmt"

	"github.com/samber/lo"

	currencyadapter "github.com/openmeterio/openmeter/openmeter/currencies/adapter"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	taxcodeadapter "github.com/openmeterio/openmeter/openmeter/taxcode/adapter"
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

	var annotations models.Annotations
	if len(sub.Annotations) > 0 {
		annotations = sub.Annotations
	} else {
		annotations = models.Annotations{}
	}

	costBasisPins := make([]subscription.CostBasisPin, 0, len(sub.Edges.CostBasisPins))
	for _, pin := range sub.Edges.CostBasisPins {
		mapped, err := FromDBSubscriptionCostBasisPin(pin)
		if err != nil {
			return subscription.Subscription{}, fmt.Errorf("mapping subscription cost basis pin: %w", err)
		}
		costBasisPins = append(costBasisPins, mapped)
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
		Annotations:     annotations,
		PlanRef:         ref,
		Name:            sub.Name,
		Description:     sub.Description,
		CustomerId:      sub.CustomerID,
		InvoiceCurrency: sub.InvoiceCurrency,
		CostBasisMode:   subscription.CostBasisMode(sub.CostBasisMode),
		CostBasisPins:   costBasisPins,
		BillingCadence:  billingCadence,
		ProRatingConfig: sub.ProRatingConfig,
		SettlementMode:  sub.SettlementMode,
		BillingAnchor:   sub.BillingAnchor.UTC(),
	}, nil
}

func FromDBSubscriptionCostBasisPin(pin *db.SubscriptionCostBasisPin) (subscription.CostBasisPin, error) {
	if pin == nil {
		return subscription.CostBasisPin{}, fmt.Errorf("unexpected nil subscription cost basis pin")
	}

	costBasis, err := pin.Edges.CostBasisOrErr()
	if err != nil {
		return subscription.CostBasisPin{}, fmt.Errorf("cost basis is not loaded: %w", err)
	}

	mapped := subscription.CostBasisPin{
		NamespacedID: models.NamespacedID{
			Namespace: pin.Namespace,
			ID:        pin.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: pin.CreatedAt.UTC(),
			UpdatedAt: pin.UpdatedAt.UTC(),
			DeletedAt: convert.SafeToUTC(pin.DeletedAt),
		},
		CustomCurrencyID: pin.CustomCurrencyID,
		InvoiceCurrency:  pin.InvoiceCurrency,
		CostBasis:        currencyadapter.FromDBCurrencyCostBasis(costBasis),
	}

	if err := mapped.Validate(); err != nil {
		return subscription.CostBasisPin{}, err
	}

	return mapped, nil
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
	if item == nil {
		return subscription.SubscriptionItem{}, fmt.Errorf("unexpected nil subscription item")
	}

	phase, err := item.Edges.PhaseOrErr()
	if err != nil {
		return subscription.SubscriptionItem{}, fmt.Errorf("failed to get phase for subscription item: %w", err)
	}

	if phase == nil {
		return subscription.SubscriptionItem{}, fmt.Errorf("unexpected nil phase for subscription item")
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

	itemCurrency, err := currencyadapter.FromDBCurrencyReference(currencyadapter.CurrencyReference{
		FiatCurrencyCode: item.FiatCurrencyCode,
		CustomCurrencyID: item.CustomCurrencyID,
		CustomCurrency:   item.Edges.CustomCurrency,
	}, true)
	if err != nil {
		return subscription.SubscriptionItem{}, fmt.Errorf("invalid subscription item currency: %w", err)
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
		UnitConfig:          item.UnitConfig,
		Key:                 item.Key,
		// NOTE: resolving feature is done on service level as there is no direct relationship between subscription items and features.
		FeatureID: nil,
		Currency:  itemCurrency,
	}

	// Map TaxCode if eagerly loaded.
	if taxCodeRow, err := item.Edges.TaxCodeOrErr(); err == nil {
		tc, err := taxcodeadapter.MapTaxCodeFromEntity(taxCodeRow)
		if err != nil {
			return subscription.SubscriptionItem{}, fmt.Errorf("invalid tax code for subscription item %s: %w", item.ID, err)
		}

		rcMeta.TaxCode = &tc
	}

	// Backfill legacy TaxConfig fields from new normalized columns.
	rcMeta.TaxConfig = productcatalog.BackfillTaxConfig(rcMeta.TaxConfig, item.TaxBehavior, rcMeta.TaxCode)

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
