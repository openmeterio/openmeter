package subscriptionaddons

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
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

func FromAPISubscriptionAddonSortField(ctx context.Context, field string) (subscriptionaddon.OrderBy, error) {
	switch field {
	case "id":
		return subscriptionaddon.OrderByID, nil
	case "created_at":
		return subscriptionaddon.OrderByCreatedAt, nil
	case "updated_at":
		return subscriptionaddon.OrderByUpdatedAt, nil
	default:
		return "", apierrors.NewUnsupportedSortFieldError(
			ctx, field, "id", "created_at", "updated_at",
		)
	}
}

func FromAPICreateSubscriptionAddonRequest(req apiv3.CreateSubscriptionAddonRequest) (subscriptionworkflow.AddAddonWorkflowInput, error) {
	timing, err := subscriptions.FromAPIBillingSubscriptionEditTiming(req.Timing)
	if err != nil {
		return subscriptionworkflow.AddAddonWorkflowInput{}, fmt.Errorf("failed to convert timing: %w", err)
	}

	meta, err := labels.ToMetadata(req.Labels)
	if err != nil {
		return subscriptionworkflow.AddAddonWorkflowInput{}, err
	}

	return subscriptionworkflow.AddAddonWorkflowInput{
		AddonID:         req.Addon.Id,
		InitialQuantity: req.Quantity,
		Timing:          timing,
		MetadataModel: models.MetadataModel{
			Metadata: meta,
		},
	}, nil
}

func toAPISubscriptionAddon(view subscription.SubscriptionView, addon subscriptionaddon.SubscriptionAddon) (apiv3.SubscriptionAddon, error) {
	now := clock.Now()

	// inst.Quantity is 0 when no instance is active at now (e.g. addon scheduled for next_billing_cycle).
	inst, _ := addon.GetInstanceAt(now)

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
			return apiv3.SubscriptionAddonRateCard{}, fmt.Errorf("failed to convert rate card: %w", err)
		}

		// JSON encoders should emit [] not null when no items are affected.
		ids := affectedMap[r.AddonRateCard.RateCard.Key()]
		if ids == nil {
			ids = []string{}
		}

		return apiv3.SubscriptionAddonRateCard{
			RateCard:                    rc,
			AffectedSubscriptionItemIds: ids,
		}, nil
	})
	if err != nil {
		return apiv3.SubscriptionAddon{}, fmt.Errorf("failed to convert rate cards: %w", err)
	}

	// Addons with no rate cards leave RateCards nil; emit [] so the response satisfies the array schema.
	if rateCards == nil {
		rateCards = []apiv3.SubscriptionAddonRateCard{}
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
