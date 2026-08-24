package subscriptions

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/plans"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromAPISubscriptionSortField(ctx context.Context, field string) (subscription.OrderBy, error) {
	switch field {
	case "id":
		return subscription.OrderByID, nil
	case "active_from":
		return subscription.OrderByActiveFrom, nil
	case "active_to":
		return subscription.OrderByActiveTo, nil
	default:
		return "", apierrors.NewUnsupportedSortFieldError(
			ctx, field, "id", "active_from", "active_to",
		)
	}
}

// subscriptionBaseFields maps the subscription's own fields — everything except the
// phases and current billing period, which are resolved from the view by
// ToAPIBillingSubscription.
func subscriptionBaseFields(sub subscription.Subscription, now time.Time) api.BillingSubscription {
	costBasisPins := make([]api.BillingSubscriptionCostBasisPin, 0, len(sub.CostBasisPins))
	for _, pin := range sub.CostBasisPins {
		costBasisPins = append(costBasisPins, api.BillingSubscriptionCostBasisPin{
			CustomCurrencyId: pin.CustomCurrencyID,
			InvoiceCurrency:  pin.InvoiceCurrency.String(),
			CostBasisId:      pin.CostBasis.ID,
		})
	}

	result := api.BillingSubscription{
		Id:              sub.ID,
		Name:            sub.Name,
		Description:     sub.Description,
		ActiveFrom:      sub.ActiveFrom,
		ActiveTo:        sub.ActiveTo,
		CustomerId:      sub.CustomerId,
		InvoiceCurrency: sub.InvoiceCurrency.String(),
		CostBasisMode:   api.BillingSubscriptionCostBasisMode(sub.CostBasisMode.OrDefault()),
		CostBasisPins:   costBasisPins,
		BillingCadence:  sub.BillingCadence.ISOString().String(),
		BillingAnchor:   sub.BillingAnchor,
		ProRatingConfig: &api.BillingSubscriptionProRatingConfig{
			Enabled: sub.ProRatingConfig.Enabled,
			Mode:    api.BillingRateCardProrationMode(sub.ProRatingConfig.Mode),
		},
		SettlementMode: lo.ToPtr(api.BillingSettlementMode(sub.SettlementMode)),
		Status:         api.BillingSubscriptionStatus(sub.GetStatusAt(now)),
		Labels:         labels.FromMetadataAnnotations(sub.Metadata, sub.Annotations),
		CreatedAt:      sub.CreatedAt,
		UpdatedAt:      sub.UpdatedAt,
		DeletedAt:      sub.DeletedAt,
	}

	// Only set if the subscription is created from a plan.
	if sub.PlanRef != nil {
		// plan_id is deprecated in favor of plan, but retained for backwards compatibility.
		result.PlanId = &sub.PlanRef.Id
		result.Plan = &api.BillingSubscriptionPlanReference{
			Id:      sub.PlanRef.Id,
			Key:     sub.PlanRef.Key,
			Version: sub.PlanRef.Version,
		}
	}

	return result
}

func ToAPIBillingSubscription(view subscription.SubscriptionView) (api.BillingSubscription, error) {
	// Resolve a single "now" and thread it through status, the current period, and
	// phase classification so every time-dependent field reflects the same instant.
	now := clock.Now()

	result := subscriptionBaseFields(view.Subscription, now)

	// The current aligned billing period only has a value while the subscription is
	// active and aligned. Querying it before the subscription starts is expected, not
	// an error, so that case leaves current_period unset.
	if period, err := view.Spec.GetAlignedBillingPeriodAt(now); err != nil {
		if !subscription.IsErrSubscriptionBillingPeriodQueriedBeforeSubscriptionStart(err) {
			return api.BillingSubscription{}, err
		}
	} else {
		result.CurrentPeriod = &api.ClosedPeriod{From: period.From, To: period.To}
	}

	phases := make([]api.BillingSubscriptionPhase, 0, len(view.Phases))
	for _, phaseView := range view.Phases {
		phase, err := toAPIBillingSubscriptionPhase(view, phaseView, now)
		if err != nil {
			return api.BillingSubscription{}, err
		}

		phases = append(phases, phase)
	}
	result.Phases = phases

	return result, nil
}

// phaseRelation classifies a phase relative to the current time, which decides which
// version of each item is surfaced.
type phaseRelation int

const (
	phaseRelationPast phaseRelation = iota
	phaseRelationCurrent
	phaseRelationFuture
)

func toAPIBillingSubscriptionPhase(view subscription.SubscriptionView, phaseView subscription.SubscriptionPhaseView, now time.Time) (api.BillingSubscriptionPhase, error) {
	phase := phaseView.SubscriptionPhase

	// A phase present on the view must have a cadence in the spec; a lookup error is a
	// spec/view inconsistency (not an open-ended phase), so surface it.
	cadence, err := view.Spec.GetPhaseCadence(phase.Key)
	if err != nil {
		return api.BillingSubscriptionPhase{}, fmt.Errorf("failed to get cadence for phase %q: %w", phase.Key, err)
	}
	activeTo := cadence.ActiveTo

	// Classify the phase relative to now to decide which item version to surface.
	relation := phaseRelationPast
	if currentPhase, ok := view.Spec.GetCurrentPhaseAt(now); ok && currentPhase.PhaseKey == phase.Key {
		relation = phaseRelationCurrent
	} else if phase.ActiveFrom.After(now) {
		relation = phaseRelationFuture
	}

	// Sort keys for a deterministic item order across responses.
	itemKeys := lo.Keys(phaseView.ItemsByKey)
	slices.Sort(itemKeys)

	items := make([]api.BillingSubscriptionItem, 0, len(itemKeys))
	for _, key := range itemKeys {
		versions := phaseView.ItemsByKey[key]
		if len(versions) == 0 {
			continue
		}

		var resolved *subscription.SubscriptionItemView
		switch relation {
		case phaseRelationCurrent:
			for i := range versions {
				if versions[i].SubscriptionItem.IsActiveAt(now) {
					resolved = &versions[i]
					break
				}
			}
		case phaseRelationFuture:
			resolved = &versions[0]
		default: // phaseRelationPast
			resolved = &versions[len(versions)-1]
		}

		if resolved == nil {
			continue
		}

		item, err := toAPIBillingSubscriptionItem(*resolved)
		if err != nil {
			return api.BillingSubscriptionPhase{}, err
		}

		items = append(items, item)
	}

	return api.BillingSubscriptionPhase{
		Id:          phase.ID,
		Key:         phase.Key,
		Name:        phase.Name,
		Description: phase.Description,
		Labels:      labels.FromMetadata(phase.Metadata),
		ActiveFrom:  phase.ActiveFrom,
		ActiveTo:    activeTo,
		CreatedAt:   phase.CreatedAt,
		UpdatedAt:   phase.UpdatedAt,
		DeletedAt:   phase.DeletedAt,
		Items:       items,
	}, nil
}

// toAPIBillingSubscriptionItem maps a resolved item version to the API item,
// reusing the shared v3 rate card converter.
func toAPIBillingSubscriptionItem(itemView subscription.SubscriptionItemView) (api.BillingSubscriptionItem, error) {
	item := itemView.SubscriptionItem

	rateCard, err := plans.ToAPIBillingRateCard(item.RateCard)
	if err != nil {
		return api.BillingSubscriptionItem{}, fmt.Errorf("failed to map rate card for item %q: %w", item.Key, err)
	}

	return api.BillingSubscriptionItem{
		Id:         item.ID,
		ActiveFrom: item.ActiveFrom,
		ActiveTo:   item.ActiveTo,
		RateCard:   rateCard,
	}, nil
}

func FromAPIBillingSubscriptionEditTimingEnum(t api.BillingSubscriptionEditTimingEnum) (subscription.Timing, error) {
	switch string(t) {
	case "immediate":
		return subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}, nil
	case "next_billing_cycle":
		return subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)}, nil
	default:
		return subscription.Timing{}, models.NewGenericValidationError(fmt.Errorf("invalid timing: %s", t))
	}
}

func FromAPIBillingSubscriptionEditTiming(t api.BillingSubscriptionEditTiming) (subscription.Timing, error) {
	// Try decoding as a custom RFC3339 datetime first, otherwise it would also decode as a "string enum"
	// and we'd never be able to distinguish enum vs datetime.
	if custom, err := t.AsDateTime(); err == nil {
		return subscription.Timing{Custom: &custom}, nil
	}

	enum, err := t.AsBillingSubscriptionEditTimingEnum()
	if err != nil {
		return subscription.Timing{}, models.NewGenericValidationError(fmt.Errorf("invalid timing"))
	}

	return FromAPIBillingSubscriptionEditTimingEnum(enum)
}

// FromAPIBillingSubscriptionCreate converts a create subscription request to a create subscription workflow input.
func FromAPIBillingSubscriptionCreate(
	namespace string,
	customerID customer.CustomerID,
	subscriptionName string,
	createSubscriptionRequest api.BillingSubscriptionCreate,
) (subscriptionworkflow.CreateSubscriptionWorkflowInput, error) {
	metadata, err := labels.ToMetadata(createSubscriptionRequest.Labels)
	if err != nil {
		return subscriptionworkflow.CreateSubscriptionWorkflowInput{}, err
	}

	workflowInput := subscriptionworkflow.CreateSubscriptionWorkflowInput{
		Namespace:     namespace,
		CustomerID:    customerID.ID,
		BillingAnchor: createSubscriptionRequest.BillingAnchor,
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Name:          subscriptionName,
			CostBasisMode: subscription.CostBasisMode(lo.FromPtr(createSubscriptionRequest.CostBasisMode)),
			Timing: subscription.Timing{
				// TODO: accept from request
				Enum: lo.ToPtr(subscription.TimingImmediate),
			},
			BillingAnchor: createSubscriptionRequest.BillingAnchor,
			MetadataModel: models.MetadataModel{
				Metadata: metadata,
			},
		},
	}

	return workflowInput, nil
}
