package subscriptionaddon

import (
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type SubscriptionAddon struct {
	models.NamespacedID
	models.ManagedModel
	models.MetadataModel

	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`

	AddonID        string `json:"addonID"`
	SubscriptionID string `json:"subscriptionID"`

	RateCards  []SubscriptionAddonRateCard                  `json:"rateCards"`
	Quantities timeutil.Timeline[SubscriptionAddonQuantity] `json:"quantities"`
}

type CreateSubscriptionAddonInput struct {
	models.MetadataModel

	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`

	AddonID        string `json:"addonID"`
	SubscriptionID string `json:"subscriptionID"`

	RateCards  []CreateSubscriptionAddonRateCardInput `json:"rateCards"`
	Quantities []CreateSubscriptionAddonQuantityInput `json:"quantities"`
}
