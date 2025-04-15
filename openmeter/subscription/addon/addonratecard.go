package subscriptionaddon

import (
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionAddonRateCard struct {
	models.NamespacedID
	models.ManagedModel

	AddonRateCard addon.RateCard `json:"addonRateCard"`

	// TODO: Remove this
	AffectedSubscriptionItemIDs []string `json:"affectedSubscriptionItemIDs"`
}

type CreateSubscriptionAddonRateCardInput struct {
	AddonRateCardID string `json:"addonRateCardID"`

	AffectedSubscriptionItemIDs []string `json:"affectedSubscriptionItemIDs"`
}
