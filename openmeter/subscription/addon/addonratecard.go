package subscriptionaddon

import (
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionAddonRateCard struct {
	models.NamespacedID
	models.ManagedModel

	AddonRateCard addon.RateCard `json:"addonRateCard"`

	AffectedSubscriptionItems []SubscriptionAddonRateCardItemRef `json:"affectedSubscriptionItems"`
}

type CreateSubscriptionAddonRateCardInput struct {
	AddonRateCardID string `json:"addonRateCardID"`

	AffectedSubscriptionItems []SubscriptionAddonRateCardItemRef `json:"affectedSubscriptionItems"`
}

type SubscriptionAddonRateCardItemRef struct {
	SubscriptionItemID        string `json:"subscriptionItemID"`
	SubscriptionItemThroughID string `json:"subscriptionItemThroughID"`
}
