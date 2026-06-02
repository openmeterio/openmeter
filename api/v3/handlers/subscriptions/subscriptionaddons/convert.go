package subscriptionaddons

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/handlers/subscriptions"
	"github.com/openmeterio/openmeter/api/v3/labels"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func toAPISubscriptionAddon(addon subscriptionaddon.SubscriptionAddon) (apiv3.SubscriptionAddon, error) {
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
	}, nil
}

func toUpdateSubscriptionAddon(addonID models.NamespacedID, body apiv3.UpdateSubscriptionAddonRequest) (subscriptionworkflow.ChangeAddonQuantityWorkflowInput, error) {
	timing, err := subscriptions.FromAPIBillingSubscriptionEditTiming(*body.Timing)
	if err != nil {
		return subscriptionworkflow.ChangeAddonQuantityWorkflowInput{}, err
	}

	return subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
		SubscriptionAddonID: addonID,
		Quantity:            lo.FromPtrOr(body.Quantity, 0),
		Timing:              timing,
	}, nil
}
