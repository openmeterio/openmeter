package subscriptionaddon

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type SubscriptionAddon struct {
	models.NamespacedID
	models.ManagedModel
	models.MetadataModel

	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`

	// AddonID        string `json:"addonID"`
	Addon          addon.Addon `json:"addon"`
	SubscriptionID string      `json:"subscriptionID"`

	RateCards  []SubscriptionAddonRateCard                  `json:"rateCards"`
	Quantities timeutil.Timeline[SubscriptionAddonQuantity] `json:"quantities"`
}

func (a SubscriptionAddon) GetInstances() []SubscriptionAddonInstance {
	quantities := a.Quantities

	// Deleted things should not get into memory but let's look out anyways
	if a.DeletedAt != nil {
		quantities = quantities.Before(*a.DeletedAt)
	}

	if len(quantities.GetTimes()) == 0 {
		return []SubscriptionAddonInstance{}
	}

	periods := quantities.GetOpenPeriods()

	if len(periods) < 1 {
		// This should never happen as len > 0
		return []SubscriptionAddonInstance{}
	}

	periods = periods[1:]

	if len(periods) != len(quantities.GetTimes()) {
		// This should never happen
		return []SubscriptionAddonInstance{}
	}

	if lo.SomeBy(periods, func(period timeutil.OpenPeriod) bool {
		return period.From == nil
	}) {
		// This should never happen
		return []SubscriptionAddonInstance{}
	}

	return lo.Map(periods, func(period timeutil.OpenPeriod, idx int) SubscriptionAddonInstance {
		quantity := quantities.GetAt(idx)

		cad, _ := models.NewCadencedModelFromPeriod(period)

		return SubscriptionAddonInstance{
			NamespacedID:  a.NamespacedID,
			ManagedModel:  a.ManagedModel,
			MetadataModel: a.MetadataModel,

			Addon:          a.Addon,
			SubscriptionID: a.SubscriptionID,

			Name:        a.Name,
			Description: a.Description,

			RateCards:     a.RateCards,
			Quantity:      quantity.GetValue().Quantity,
			CadencedModel: cad,
		}
	})
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
