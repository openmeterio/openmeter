package subscriptionaddon

import (
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/models"
)

// SubscriptionAddonInstance is a "virtual" instance of a SubscriptionAddon:
// It merges the quantity information with the Addon information itself and represents the "effective" value of the addon for a given period.
type SubscriptionAddonInstance struct {
	models.NamespacedID
	models.ManagedModel
	models.MetadataModel
	models.CadencedModel

	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`

	// AddonID        string `json:"addonID"`
	Addon          addon.Addon `json:"addon"`
	SubscriptionID string      `json:"subscriptionID"`

	RateCards []SubscriptionAddonRateCard `json:"rateCards"`
	Quantity  int                         `json:"quantity"`
}
