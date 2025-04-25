package httpdriver

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	subscriptionhttp "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/http"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	addondiff "github.com/openmeterio/openmeter/openmeter/subscription/addon/diff"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func MapCreateSubscriptionAddonRequestToInput(req api.SubscriptionAddonCreate) (subscriptionworkflow.AddAddonWorkflowInput, error) {
	timing, err := subscriptionhttp.MapAPITimingToTiming(req.Timing)
	if err != nil {
		return subscriptionworkflow.AddAddonWorkflowInput{}, fmt.Errorf("failed to cast Timing: %w", err)
	}

	r := subscriptionworkflow.AddAddonWorkflowInput{
		AddonID:         req.Addon.Id,
		InitialQuantity: req.Quantity,
		Timing:          timing,
	}

	if req.Metadata != nil {
		r.MetadataModel.Metadata = lo.FromPtr(req.Metadata)
	}

	return r, nil
}

func MapSubscriptionAddonToResponse(view subscription.SubscriptionView, addon subscriptionaddon.SubscriptionAddon) (api.SubscriptionAddon, error) {
	now := clock.Now()

	// If instance is not found, quantity will be 0
	inst, _ := addon.GetInstanceAt(now)

	pers := lo.Map(addon.GetInstances(), func(i subscriptionaddon.SubscriptionAddonInstance, _ int) timeutil.OpenPeriod {
		return i.AsPeriod()
	})

	if len(pers) == 0 {
		return api.SubscriptionAddon{}, errors.New("no instances found")
	}

	union := lo.Reduce(pers, func(agg timeutil.OpenPeriod, item timeutil.OpenPeriod, _ int) timeutil.OpenPeriod {
		return agg.Union(item)
	}, pers[0])

	affectedMap := addondiff.GetAffectedItemIDs(view, addon)

	rateCards, err := slicesx.MapWithErr(addon.RateCards, func(r subscriptionaddon.SubscriptionAddonRateCard) (api.SubscriptionAddonRateCard, error) {
		rc, err := productcataloghttp.FromRateCard(r.AddonRateCard.RateCard)
		if err != nil {
			return api.SubscriptionAddonRateCard{}, fmt.Errorf("failed to cast RateCard: %w", err)
		}

		ids := affectedMap[r.AddonRateCard.RateCard.Key()]

		return api.SubscriptionAddonRateCard{
			RateCard:                    rc,
			AffectedSubscriptionItemIds: ids,
		}, nil
	})
	if err != nil {
		return api.SubscriptionAddon{}, fmt.Errorf("failed to cast RateCards: %w", err)
	}

	return api.SubscriptionAddon{
		Id:             addon.ID,
		CreatedAt:      addon.CreatedAt,
		UpdatedAt:      addon.UpdatedAt,
		DeletedAt:      addon.DeletedAt,
		Metadata:       lo.EmptyableToPtr(api.Metadata(addon.Metadata)),
		Description:    addon.Description,
		Name:           addon.Name,
		SubscriptionId: addon.SubscriptionID,
		Addon: struct {
			Id           string                "json:\"id\""
			InstanceType api.AddonInstanceType "json:\"instanceType\""
			Key          string                "json:\"key\""
			Version      int                   "json:\"version\""
		}{
			Id:           addon.Addon.ID,
			InstanceType: api.AddonInstanceType(addon.Addon.InstanceType),
			Key:          addon.Addon.Key,
			Version:      addon.Addon.Version,
		},
		ActiveFrom: *union.From,
		ActiveTo:   union.To,
		Quantity:   inst.Quantity,
		QuantityAt: now,
		Timeline: lo.Map(addon.GetInstances(), func(i subscriptionaddon.SubscriptionAddonInstance, _ int) api.SubscriptionAddonTimelineSegment {
			return api.SubscriptionAddonTimelineSegment{
				Quantity:   i.Quantity,
				ActiveFrom: i.CadencedModel.ActiveFrom,
				ActiveTo:   i.CadencedModel.ActiveTo,
			}
		}),
		RateCards: rateCards,
	}, nil
}
