package subscriptionaddon

import (
	"errors"
	"fmt"

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

	AddonID        string `json:"addonID"`
	SubscriptionID string `json:"subscriptionID"`

	RateCards       []CreateSubscriptionAddonRateCardInput `json:"rateCards"`
	InitialQuantity CreateSubscriptionAddonQuantityInput   `json:"initialQuantity"`
}

func (i CreateSubscriptionAddonInput) Validate() error {
	var errs []error

	if i.AddonID == "" {
		errs = append(errs, errors.New("addonID is required"))
	}

	if i.SubscriptionID == "" {
		errs = append(errs, errors.New("subscriptionID is required"))
	}

	if len(i.RateCards) == 0 {
		errs = append(errs, errors.New("rateCards weren't provided"))
	}

	if err := i.InitialQuantity.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("initialQuantity: %w", err))
	}

	if i.InitialQuantity.Quantity == 0 {
		errs = append(errs, errors.New("initialQuantity.quantity cannot be 0"))
	}

	return errors.Join(errs...)
}
