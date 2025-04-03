package subscriptionaddonrepo

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// MapSubscriptionAddon maps a db.SubscriptionAddon to a subscriptionaddon.SubscriptionAddon
func MapSubscriptionAddon(
	entity *db.SubscriptionAddon,
) subscriptionaddon.SubscriptionAddon {
	base := subscriptionaddon.SubscriptionAddon{
		NamespacedID: models.NamespacedID{
			ID:        entity.ID,
			Namespace: entity.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		MetadataModel: models.MetadataModel{
			Metadata: entity.Metadata,
		},
		// Name and Description should be populated but don't exist in the schema yet
		Name:           "",
		Description:    nil,
		AddonID:        entity.AddonID,
		SubscriptionID: entity.SubscriptionID,
	}

	if len(entity.Edges.Quantities) > 0 {
		quantities := MapSubscriptionAddonQuantities(entity.Edges.Quantities)
		base.Quantities = timeutil.NewTimeline[subscriptionaddon.SubscriptionAddonQuantity](lo.Map(quantities, func(q subscriptionaddon.SubscriptionAddonQuantity, _ int) timeutil.Timed[subscriptionaddon.SubscriptionAddonQuantity] {
			return q.AsTimed()
		}))
	}

	if len(entity.Edges.RateCards) > 0 {
		base.RateCards = MapSubscriptionAddonRateCards(entity.Edges.RateCards)
	}

	return base
}

// MapSubscriptionAddons maps a slice of db.SubscriptionAddon to a slice of subscriptionaddon.SubscriptionAddon
func MapSubscriptionAddons(entities []*db.SubscriptionAddon) []subscriptionaddon.SubscriptionAddon {
	return lo.Map(entities, func(entity *db.SubscriptionAddon, _ int) subscriptionaddon.SubscriptionAddon {
		return MapSubscriptionAddon(entity)
	})
}

// MapSubscriptionAddonRateCard maps a db.SubscriptionAddonRateCard to a subscriptionaddon.SubscriptionAddonRateCard
func MapSubscriptionAddonRateCard(entity *db.SubscriptionAddonRateCard) subscriptionaddon.SubscriptionAddonRateCard {
	base := subscriptionaddon.SubscriptionAddonRateCard{
		NamespacedID: models.NamespacedID{
			ID:        entity.ID,
			Namespace: entity.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		RateCardID: entity.ID, // Using ID as RateCardID for now
	}
	if len(entity.Edges.Items) > 0 {
		base.AffectedSubscriptionItemIDs = lo.Map(entity.Edges.Items, func(item *db.SubscriptionAddonRateCardItemLink, _ int) string {
			return item.SubscriptionItemID
		})
	}

	return base
}

// MapSubscriptionAddonRateCards maps a slice of db.SubscriptionAddonRateCard to a slice of subscriptionaddon.SubscriptionAddonRateCard
func MapSubscriptionAddonRateCards(entities []*db.SubscriptionAddonRateCard) []subscriptionaddon.SubscriptionAddonRateCard {
	return lo.Map(entities, func(entity *db.SubscriptionAddonRateCard, _ int) subscriptionaddon.SubscriptionAddonRateCard {
		return MapSubscriptionAddonRateCard(entity)
	})
}

// MapSubscriptionAddonQuantity maps a db.SubscriptionAddonQuantity to a subscriptionaddon.SubscriptionAddonQuantity
func MapSubscriptionAddonQuantity(entity *db.SubscriptionAddonQuantity) subscriptionaddon.SubscriptionAddonQuantity {
	return subscriptionaddon.SubscriptionAddonQuantity{
		NamespacedID: models.NamespacedID{
			ID:        entity.ID,
			Namespace: entity.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		ActiveFrom: entity.ActiveFrom,
		Quantity:   entity.Quantity,
	}
}

// MapSubscriptionAddonQuantities maps a slice of db.SubscriptionAddonQuantity to a slice of subscriptionaddon.SubscriptionAddonQuantity
func MapSubscriptionAddonQuantities(entities []*db.SubscriptionAddonQuantity) []subscriptionaddon.SubscriptionAddonQuantity {
	quantities := make([]subscriptionaddon.SubscriptionAddonQuantity, len(entities))
	for i, entity := range entities {
		quantities[i] = MapSubscriptionAddonQuantity(entity)
	}
	return quantities
}
