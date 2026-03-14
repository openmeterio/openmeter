package adapter

import (
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func MapChargeFromDB(entity *entdb.Charge) meta.Charge {
	return meta.Charge{
		ManagedResource: MapManagedResourceFromDB(entity),
		Intent:          MapIntentFromDB(entity),
		Type:            entity.Type,
		Status:          entity.Status,
	}
}

// MapManagedResourceFromDB extracts the ManagedResource from a DB Charge entity.
func MapManagedResourceFromDB(entity *entdb.Charge) meta.ManagedResource {
	return meta.ManagedResource{
		NamespacedModel: models.NamespacedModel{
			Namespace: entity.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		ID: entity.ID,
	}
}

// MapIntentFromDB extracts the IntentMeta from a DB Charge entity.
func MapIntentFromDB(entity *entdb.Charge) meta.Intent {
	return meta.Intent{
		Name:        entity.Name,
		Metadata:    entity.Metadata,
		Annotations: entity.Annotations,
		ManagedBy:   entity.ManagedBy,
		CustomerID:  entity.CustomerID,
		Currency:    entity.Currency,
		ServicePeriod: timeutil.ClosedPeriod{
			From: entity.ServicePeriodFrom.UTC(),
			To:   entity.ServicePeriodTo.UTC(),
		},
		FullServicePeriod: timeutil.ClosedPeriod{
			From: entity.FullServicePeriodFrom.UTC(),
			To:   entity.FullServicePeriodTo.UTC(),
		},
		BillingPeriod: timeutil.ClosedPeriod{
			From: entity.BillingPeriodFrom.UTC(),
			To:   entity.BillingPeriodTo.UTC(),
		},
		UniqueReferenceID: entity.UniqueReferenceID,
		Subscription:      mapSubscriptionRefFromDB(entity),
	}
}

// mapSubscriptionRefFromDB extracts a SubscriptionReference from a DB Charge entity, returning nil if any ID is missing.
func mapSubscriptionRefFromDB(entity *entdb.Charge) *meta.SubscriptionReference {
	if entity.SubscriptionID == nil || entity.SubscriptionPhaseID == nil || entity.SubscriptionItemID == nil {
		return nil
	}

	return &meta.SubscriptionReference{
		SubscriptionID: *entity.SubscriptionID,
		PhaseID:        *entity.SubscriptionPhaseID,
		ItemID:         *entity.SubscriptionItemID,
	}
}
