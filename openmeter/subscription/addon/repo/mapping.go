package subscriptionaddonrepo

import (
	"errors"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	addonrepo "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/adapter"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// MapSubscriptionAddon maps a db.SubscriptionAddon to a subscriptionaddon.SubscriptionAddon
func MapSubscriptionAddon(
	entity *db.SubscriptionAddon,
) (subscriptionaddon.SubscriptionAddon, error) {
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
		SubscriptionID: entity.SubscriptionID,
	}

	if entity.Edges.Addon != nil {
		dbAdd := entity.Edges.Addon
		base.Name = dbAdd.Name
		base.Description = dbAdd.Description

		add, err := addonrepo.FromAddonRow(*dbAdd)
		if err != nil {
			return subscriptionaddon.SubscriptionAddon{}, err
		}
		base.Addon = *add
	}

	if len(entity.Edges.Quantities) > 0 {
		quantities := MapSubscriptionAddonQuantities(entity.Edges.Quantities)
		base.Quantities = timeutil.NewTimeline[subscriptionaddon.SubscriptionAddonQuantity](lo.Map(quantities, func(q subscriptionaddon.SubscriptionAddonQuantity, _ int) timeutil.Timed[subscriptionaddon.SubscriptionAddonQuantity] {
			return q.AsTimed()
		}))
	}

	if len(entity.Edges.RateCards) > 0 {
		rateCards, err := MapSubscriptionAddonRateCards(entity.Edges.RateCards)
		if err != nil {
			return subscriptionaddon.SubscriptionAddon{}, err
		}
		base.RateCards = rateCards
	}

	return base, nil
}

// MapSubscriptionAddons maps a slice of db.SubscriptionAddon to a slice of subscriptionaddon.SubscriptionAddon
func MapSubscriptionAddons(entities []*db.SubscriptionAddon) ([]subscriptionaddon.SubscriptionAddon, error) {
	return slicesx.MapWithErr(entities, func(entity *db.SubscriptionAddon) (subscriptionaddon.SubscriptionAddon, error) {
		return MapSubscriptionAddon(entity)
	})
}

// MapSubscriptionAddonRateCard maps a db.SubscriptionAddonRateCard to a subscriptionaddon.SubscriptionAddonRateCard
func MapSubscriptionAddonRateCard(entity *db.SubscriptionAddonRateCard) (subscriptionaddon.SubscriptionAddonRateCard, error) {
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
	}
	if len(entity.Edges.Items) > 0 {
		base.AffectedSubscriptionItemIDs = lo.Map(entity.Edges.Items, func(item *db.SubscriptionAddonRateCardItemLink, _ int) string {
			return item.SubscriptionItemID
		})
	}

	if entity.Edges.AddonRatecard != nil {
		arc, err := addonrepo.FromAddonRateCardRow(*entity.Edges.AddonRatecard)
		if err != nil {
			return subscriptionaddon.SubscriptionAddonRateCard{}, err
		}
		if arc == nil {
			return subscriptionaddon.SubscriptionAddonRateCard{}, errors.New("invalid addon rate card")
		}
		base.AddonRateCard = *arc
	}

	return base, nil
}

// MapSubscriptionAddonRateCards maps a slice of db.SubscriptionAddonRateCard to a slice of subscriptionaddon.SubscriptionAddonRateCard
func MapSubscriptionAddonRateCards(entities []*db.SubscriptionAddonRateCard) ([]subscriptionaddon.SubscriptionAddonRateCard, error) {
	return slicesx.MapWithErr(entities, func(entity *db.SubscriptionAddonRateCard) (subscriptionaddon.SubscriptionAddonRateCard, error) {
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
