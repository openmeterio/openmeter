package subscriptionaddons

import (
	"errors"

	"github.com/samber/lo"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func toAPISubscriptionAddon(addon subscriptionaddon.SubscriptionAddon) (apiv3.SubscriptionAddon, error) {
	now := clock.Now()

	// If no instance is active at `now`, quantity stays 0.
	inst, _ := addon.GetInstanceAt(now)

	pers := lo.Map(addon.GetInstances(), func(i subscriptionaddon.SubscriptionAddonInstance, _ int) timeutil.OpenPeriod {
		return i.AsPeriod()
	})

	if len(pers) == 0 {
		return apiv3.SubscriptionAddon{}, errors.New("no instances found for subscription addon")
	}

	union := lo.Reduce(pers, func(agg timeutil.OpenPeriod, item timeutil.OpenPeriod, _ int) timeutil.OpenPeriod {
		return agg.Union(item)
	}, pers[0])

	return apiv3.SubscriptionAddon{
		Id:          addon.ID,
		Name:        addon.Name,
		Description: addon.Description,
		CreatedAt:   addon.CreatedAt,
		UpdatedAt:   addon.UpdatedAt,
		DeletedAt:   addon.DeletedAt,
		Addon: apiv3.AddonReferenceItem{
			Id: addon.Addon.ID,
		},
		Quantity:   inst.Quantity,
		QuantityAt: now,
		ActiveFrom: lo.FromPtrOr(union.From, now),
		ActiveTo:   union.To,
	}, nil
}
