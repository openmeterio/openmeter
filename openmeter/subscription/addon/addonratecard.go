package subscriptionaddon

import (
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionAddonRateCard struct {
	models.NamespacedID
	models.ManagedModel

	AddonRateCard addon.RateCard `json:"addonRateCard"`

	AffectedSubscriptionItemIDs []string `json:"affectedSubscriptionItemIDs"`
}

type CreateSubscriptionAddonRateCardInput struct {
	RateCardID string `json:"rateCardID"`

	AffectedSubscriptionItemIDs []string `json:"affectedSubscriptionItemIDs"`
}
