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
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
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

// ToAPIBillingSubscriptionBase maps a bare subscription — no view — to the
// API model: base fields only, an empty phase list, and no current period.
// It serves callers that side-load subscriptions without their specs (e.g.
// the customer charges subscription expand); the view-derived fields need
// ToAPIBillingSubscription.
func ToAPIBillingSubscriptionBase(sub subscription.Subscription) api.BillingSubscription {
	result := subscriptionBaseFields(sub, clock.Now())
	result.Phases = []api.BillingSubscriptionPhase{}

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

// FromAPIBillingSubscriptionEditOperation maps a single API edit operation to the
// domain subscription patch that applies it to the target spec. The mapping mirrors
// the v1 editSubscription handler; rate cards are converted via the shared plans
// converter so add-item stays consistent with plan/subscription creation.
func FromAPIBillingSubscriptionEditOperation(op api.BillingSubscriptionEditOperation) (subscription.Patch, error) {
	disc, err := op.Discriminator()
	if err != nil {
		return nil, models.NewGenericValidationError(fmt.Errorf("failed to read edit operation type: %w", err))
	}

	switch disc {
	case string(api.BillingSubscriptionEditAddItemTypeAddItem):
		apiP, err := op.AsBillingSubscriptionEditAddItem()
		if err != nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("failed to decode add_item operation: %w", err))
		}

		rc, err := plans.FromAPIBillingRateCard(apiP.RateCard)
		if err != nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("failed to convert add_item rate card: %w", err))
		}

		phaseRC := &plansubscription.RateCard{
			PhaseKey: apiP.PhaseKey,
			RateCard: rc,
		}

		return patch.PatchAddItem{
			PhaseKey: apiP.PhaseKey,
			ItemKey:  rc.Key(),
			CreateInput: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput:     phaseRC.ToCreateSubscriptionItemPlanInput(),
					CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{},
				},
			},
		}, nil

	case string(api.BillingSubscriptionEditRemoveItemTypeRemoveItem):
		apiP, err := op.AsBillingSubscriptionEditRemoveItem()
		if err != nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("failed to decode remove_item operation: %w", err))
		}

		return patch.PatchRemoveItem{
			PhaseKey: apiP.PhaseKey,
			ItemKey:  apiP.ItemKey,
		}, nil

	case string(api.BillingSubscriptionEditAddPhaseTypeAddPhase):
		apiP, err := op.AsBillingSubscriptionEditAddPhase()
		if err != nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("failed to decode add_phase operation: %w", err))
		}

		// start_after is required but nullable: a null (or absent) value leaves the
		// zero offset, i.e. the subscription start. The domain then rejects that as
		// "in the past" on a running subscription, matching v1's null handling.
		var startAfter datetime.ISODuration
		if apiP.Phase.StartAfter.IsSpecified() && !apiP.Phase.StartAfter.IsNull() {
			sa, err := apiP.Phase.StartAfter.Get()
			if err != nil {
				return nil, models.NewGenericValidationError(fmt.Errorf("failed to read add_phase start_after: %w", err))
			}
			startAfter, err = datetime.ISODurationString(sa).Parse()
			if err != nil {
				return nil, models.NewGenericValidationError(fmt.Errorf("failed to parse add_phase start_after: %w", err))
			}
		}

		var duration *datetime.ISODuration
		if apiP.Phase.Duration != nil {
			d, err := datetime.ISODurationString(*apiP.Phase.Duration).Parse()
			if err != nil {
				return nil, models.NewGenericValidationError(fmt.Errorf("failed to parse add_phase duration: %w", err))
			}
			duration = &d
		}

		return patch.PatchAddPhase{
			PhaseKey: apiP.Phase.Key,
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				Duration: duration,
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:    apiP.Phase.Key,
					StartAfter:  startAfter,
					Name:        apiP.Phase.Name,
					Description: apiP.Phase.Description,
				},
				CreateSubscriptionPhaseCustomerInput: subscription.CreateSubscriptionPhaseCustomerInput{},
			},
		}, nil

	case string(api.BillingSubscriptionEditRemovePhaseTypeRemovePhase):
		apiP, err := op.AsBillingSubscriptionEditRemovePhase()
		if err != nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("failed to decode remove_phase operation: %w", err))
		}

		var shift subscription.RemoveSubscriptionPhaseShifting
		switch apiP.Shift {
		case api.BillingSubscriptionRemovePhaseShiftingNext:
			shift = subscription.RemoveSubscriptionPhaseShiftNext
		case api.BillingSubscriptionRemovePhaseShiftingPrev:
			shift = subscription.RemoveSubscriptionPhaseShiftPrev
		default:
			return nil, models.NewGenericValidationError(fmt.Errorf("invalid remove_phase shift: %s", apiP.Shift))
		}

		return patch.PatchRemovePhase{
			PhaseKey: apiP.PhaseKey,
			RemoveInput: subscription.RemoveSubscriptionPhaseInput{
				Shift: shift,
			},
		}, nil

	case string(api.BillingSubscriptionEditStretchPhaseTypeStretchPhase):
		apiP, err := op.AsBillingSubscriptionEditStretchPhase()
		if err != nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("failed to decode stretch_phase operation: %w", err))
		}

		d, err := datetime.ISODurationString(apiP.ExtendBy).Parse()
		if err != nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("failed to parse stretch_phase extend_by: %w", err))
		}

		return patch.PatchStretchPhase{
			PhaseKey: apiP.PhaseKey,
			Duration: d,
		}, nil

	case string(api.BillingSubscriptionEditUnscheduleEditTypeUnscheduleEdit):
		return patch.PatchUnscheduleEdit{}, nil

	default:
		return nil, models.NewGenericValidationError(fmt.Errorf("unknown edit operation type: %s", disc))
	}
}

// FromAPIBillingSubscriptionCustomPlan maps an inline (custom) plan definition from a
// create or change request into the domain plan create input. It performs pure
// mapping only: plan validity, currency resolution, and the fiat/custom-currency
// rules are enforced downstream by the subscription service and spec validation.
func FromAPIBillingSubscriptionCustomPlan(namespace string, body api.BillingSubscriptionCustomPlan) (plan.CreatePlanInput, error) {
	metadata, err := labels.ToMetadata(body.Labels)
	if err != nil {
		return plan.CreatePlanInput{}, fmt.Errorf("failed to convert label metadata: %w", err)
	}

	req := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:            body.Name,
				Description:     body.Description,
				Metadata:        metadata,
				ProRatingConfig: plans.ToProRatingConfig(body.ProRatingEnabled),
			},
		},
	}

	// currencyx.Code (not FiatCode) so custom currencies are accepted; the resolver
	// and spec validation reject unregistered or otherwise invalid currencies later.
	req.Currency = currencies.NewCurrencyReference(currencyx.Code(body.Currency))

	billingCadence, err := datetime.ISODurationString(body.BillingCadence).Parse()
	if err != nil {
		return req, fmt.Errorf("invalid billing cadence: %w", err)
	}
	req.BillingCadence = billingCadence

	if len(body.Phases) > 0 {
		req.Phases = make([]productcatalog.Phase, 0, len(body.Phases))
		for _, phase := range body.Phases {
			p, err := plans.FromAPIBillingPlanPhase(phase)
			if err != nil {
				return req, fmt.Errorf("failed to convert phase: %w", err)
			}
			req.Phases = append(req.Phases, p)
		}
	}

	return req, nil
}
