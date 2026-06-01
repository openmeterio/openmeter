package subscriptionaddons

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/handlers/plans"
	"github.com/openmeterio/openmeter/api/v3/handlers/subscriptions"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	addondiff "github.com/openmeterio/openmeter/openmeter/subscription/addon/diff"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func mapCreateSubscriptionAddonRequestToInput(req apiv3.CreateSubscriptionAddonRequest) (subscriptionworkflow.AddAddonWorkflowInput, error) {
	timing, err := subscriptions.FromAPIBillingSubscriptionEditTiming(req.Timing)
	if err != nil {
		return subscriptionworkflow.AddAddonWorkflowInput{}, fmt.Errorf("failed to cast Timing: %w", err)
	}

	meta, err := labels.ToMetadata(req.Labels)
	if err != nil {
		return subscriptionworkflow.AddAddonWorkflowInput{}, err
	}

	r := subscriptionworkflow.AddAddonWorkflowInput{
		AddonID:         req.Addon.Id,
		InitialQuantity: req.Quantity,
		Timing:          timing,
		MetadataModel: models.MetadataModel{
			Metadata: meta,
		},
	}

	return r, nil
}

func toAPISubscriptionAddon(view subscription.SubscriptionView, addon subscriptionaddon.SubscriptionAddon) (apiv3.SubscriptionAddon, error) {
	now := clock.Now()

	inst, found := addon.GetInstanceAt(now)
	if !found {
		return apiv3.SubscriptionAddon{}, models.NewGenericNotFoundError(fmt.Errorf("no instance is active at %s", now.Format(time.RFC3339)))
	}

	pers := lo.Map(addon.GetInstances(), func(i subscriptionaddon.SubscriptionAddonInstance, _ int) timeutil.OpenPeriod {
		return i.AsPeriod()
	})

	if len(pers) == 0 {
		return apiv3.SubscriptionAddon{}, models.NewGenericNotFoundError(errors.New("no instances found for subscription addon"))
	}

	union := lo.Reduce(pers, func(agg timeutil.OpenPeriod, item timeutil.OpenPeriod, _ int) timeutil.OpenPeriod {
		return agg.Union(item)
	}, pers[0])

	affectedMap := addondiff.GetAffectedItemIDs(view, addon)

	rateCards, err := slicesx.MapWithErr(addon.RateCards, func(r subscriptionaddon.SubscriptionAddonRateCard) (apiv3.SubscriptionAddonRateCard, error) {
		rc, err := plans.ToAPIBillingRateCard(r.AddonRateCard.RateCard)
		if err != nil {
			return apiv3.SubscriptionAddonRateCard{}, fmt.Errorf("failed to cast RateCard: %w", err)
		}

		ids := affectedMap[r.AddonRateCard.RateCard.Key()]

		return apiv3.SubscriptionAddonRateCard{
			RateCard:                    rc,
			AffectedSubscriptionItemIds: ids,
		}, nil
	})
	if err != nil {
		return apiv3.SubscriptionAddon{}, fmt.Errorf("failed to cast RateCards: %w", err)
	}

	return apiv3.SubscriptionAddon{
		Id:          addon.ID,
		Name:        addon.Name,
		Description: addon.Description,
		CreatedAt:   addon.CreatedAt,
		UpdatedAt:   addon.UpdatedAt,
		DeletedAt:   addon.DeletedAt,
		Addon: apiv3.AddonReference{
			Id: addon.Addon.ID,
		},
		Labels:     labels.FromMetadata(addon.Metadata),
		Quantity:   inst.Quantity,
		QuantityAt: now,
		ActiveFrom: lo.FromPtrOr(union.From, now),
		ActiveTo:   union.To,
		Timeline: lo.Map(addon.GetInstances(), func(i subscriptionaddon.SubscriptionAddonInstance, _ int) apiv3.SubscriptionAddonTimelineSegment {
			return apiv3.SubscriptionAddonTimelineSegment{
				Quantity:   i.Quantity,
				ActiveFrom: i.CadencedModel.ActiveFrom,
				ActiveTo:   i.CadencedModel.ActiveTo,
			}
		}),
		RateCards: rateCards,
	}, nil
}
