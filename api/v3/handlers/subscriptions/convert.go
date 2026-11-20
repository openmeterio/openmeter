package subscriptions

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/plans"
	"github.com/openmeterio/openmeter/api/v3/labels"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
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
func subscriptionBaseFields(sub subscription.Subscription, now time.Time) apiv3models.BillingSubscriptionBase {
	costBasisPins := make([]apiv3models.BillingSubscriptionCostBasisPin, 0, len(sub.CostBasisPins))
	for _, pin := range sub.CostBasisPins {
		costBasisPins = append(costBasisPins, apiv3models.BillingSubscriptionCostBasisPin{
			CustomCurrencyId: pin.CustomCurrencyID,
			InvoiceCurrency:  pin.InvoiceCurrency.String(),
			CostBasisId:      pin.CostBasis.ID,
		})
	}

	result := apiv3models.BillingSubscriptionBase{
		Id:              sub.ID,
		Name:            sub.Name,
		Description:     sub.Description,
		ActiveFrom:      sub.ActiveFrom,
		ActiveTo:        sub.ActiveTo,
		CustomerId:      sub.CustomerId,
		InvoiceCurrency: sub.InvoiceCurrency.String(),
		CostBasisMode:   apiv3models.BillingSubscriptionCostBasisMode(sub.CostBasisMode.OrDefault()),
		CostBasisPins:   costBasisPins,
		BillingCadence:  sub.BillingCadence.ISOString().String(),
		BillingAnchor:   sub.BillingAnchor,
		BillingStartsAt: sub.BillingStartsAt,
		ProRatingConfig: &apiv3models.BillingSubscriptionProRatingConfig{
			Enabled: sub.ProRatingConfig.Enabled,
			Mode:    apiv3models.BillingRateCardProrationMode(sub.ProRatingConfig.Mode),
		},
		SettlementMode: lo.ToPtr(apiv3models.BillingSettlementMode(sub.SettlementMode)),
		Status:         apiv3models.BillingSubscriptionStatus(sub.GetStatusAt(now)),
		Labels:         labels.FromMetadataAnnotations(sub.Metadata, sub.Annotations),
		CreatedAt:      sub.CreatedAt,
		UpdatedAt:      sub.UpdatedAt,
		DeletedAt:      sub.DeletedAt,
	}

	// Only set if the subscription is created from a plan.
	if sub.PlanRef != nil {
		// plan_id is deprecated in favor of plan, but retained for backwards compatibility.
		result.PlanId = &sub.PlanRef.Id
		result.Plan = &apiv3models.BillingSubscriptionPlanReference{
			Id:      sub.PlanRef.Id,
			Key:     sub.PlanRef.Key,
			Version: sub.PlanRef.Version,
		}
	}

	return result
}

// ToAPIBillingSubscriptionWithoutView maps a bare subscription — no view — to the
// API model: base fields only, an empty phase list, and no current period.
// It serves callers that side-load subscriptions without their specs (e.g.
// the customer charges subscription expand); the view-derived fields need
// ToAPIBillingSubscription.
func ToAPIBillingSubscriptionWithoutView(sub subscription.Subscription) apiv3models.BillingSubscription {
	return mapSubscriptionBaseToFull(subscriptionBaseFields(sub, clock.Now()))
}

// ToAPIBillingSubscriptionBase maps the subscription's own fields without view data.
func ToAPIBillingSubscriptionBase(sub subscription.Subscription) apiv3models.BillingSubscriptionBase {
	return subscriptionBaseFields(sub, clock.Now())
}

// The generated full model is flat, so its shared fields need an explicit mapping.
func mapSubscriptionBaseToFull(fields apiv3models.BillingSubscriptionBase) apiv3models.BillingSubscription {
	return apiv3models.BillingSubscription{
		Phases:          []apiv3models.BillingSubscriptionPhase{},
		Id:              fields.Id,
		Name:            fields.Name,
		Description:     fields.Description,
		ActiveFrom:      fields.ActiveFrom,
		ActiveTo:        fields.ActiveTo,
		CustomerId:      fields.CustomerId,
		InvoiceCurrency: fields.InvoiceCurrency,
		CostBasisMode:   fields.CostBasisMode,
		CostBasisPins:   fields.CostBasisPins,
		BillingCadence:  fields.BillingCadence,
		BillingAnchor:   fields.BillingAnchor,
		BillingStartsAt: fields.BillingStartsAt,
		ProRatingConfig: fields.ProRatingConfig,
		SettlementMode:  fields.SettlementMode,
		Status:          fields.Status,
		Labels:          fields.Labels,
		CreatedAt:       fields.CreatedAt,
		UpdatedAt:       fields.UpdatedAt,
		DeletedAt:       fields.DeletedAt,
		PlanId:          fields.PlanId,
		Plan:            fields.Plan,
	}
}

func ToAPIBillingSubscription(view subscription.SubscriptionView) (apiv3models.BillingSubscription, error) {
	// Resolve a single "now" and thread it through status, the current period, and
	// phase classification so every time-dependent field reflects the same instant.
	now := clock.Now()

	result := mapSubscriptionBaseToFull(subscriptionBaseFields(view.Subscription, now))

	// The current aligned billing period only has a value while the subscription is
	// active and aligned. Querying it before the subscription starts is expected, not
	// an error, so that case leaves current_period unset.
	if period, err := view.Spec.GetAlignedBillingPeriodAt(now); err != nil {
		if !subscription.IsErrSubscriptionBillingPeriodQueriedBeforeSubscriptionStart(err) {
			return apiv3models.BillingSubscription{}, err
		}
	} else {
		result.CurrentPeriod = &apiv3models.ClosedPeriod{From: period.From, To: period.To}
	}

	phases := make([]apiv3models.BillingSubscriptionPhase, 0, len(view.Phases))
	for _, phaseView := range view.Phases {
		phase, err := toAPIBillingSubscriptionPhase(view, phaseView, now)
		if err != nil {
			return apiv3models.BillingSubscription{}, err
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

func toAPIBillingSubscriptionPhase(view subscription.SubscriptionView, phaseView subscription.SubscriptionPhaseView, now time.Time) (apiv3models.BillingSubscriptionPhase, error) {
	phase := phaseView.SubscriptionPhase

	// A phase present on the view must have a cadence in the spec; a lookup error is a
	// spec/view inconsistency (not an open-ended phase), so surface it.
	cadence, err := view.Spec.GetPhaseCadence(phase.Key)
	if err != nil {
		return apiv3models.BillingSubscriptionPhase{}, fmt.Errorf("failed to get cadence for phase %q: %w", phase.Key, err)
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

	items := make([]apiv3models.BillingSubscriptionItem, 0, len(itemKeys))
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
			return apiv3models.BillingSubscriptionPhase{}, err
		}

		items = append(items, item)
	}

	return apiv3models.BillingSubscriptionPhase{
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
func toAPIBillingSubscriptionItem(itemView subscription.SubscriptionItemView) (apiv3models.BillingSubscriptionItem, error) {
	item := itemView.SubscriptionItem

	rateCard, err := plans.ToAPIBillingRateCard(item.RateCard)
	if err != nil {
		return apiv3models.BillingSubscriptionItem{}, fmt.Errorf("failed to map rate card for item %q: %w", item.Key, err)
	}

	return apiv3models.BillingSubscriptionItem{
		Id:         item.ID,
		ActiveFrom: item.ActiveFrom,
		ActiveTo:   item.ActiveTo,
		RateCard:   rateCard,
	}, nil
}

func FromAPIBillingSubscriptionEditTimingEnum(t apiv3models.BillingSubscriptionEditTimingEnum) (subscription.Timing, error) {
	switch string(t) {
	case "immediate":
		return subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}, nil
	case "next_billing_cycle":
		return subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)}, nil
	default:
		return subscription.Timing{}, models.NewGenericValidationError(fmt.Errorf("invalid timing: %s", t))
	}
}

func FromAPIBillingSubscriptionEditTiming(t apiv3models.BillingSubscriptionEditTiming) (subscription.Timing, error) {
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

// FromAPIBillingSubscriptionCreateTiming maps the create-body timing to the domain
// timing. Create timing is deliberately narrower than edit timing: only immediate
// or a custom timestamp — next_billing_cycle is not a create value (a new
// subscription has no current billing cycle), so it is not part of the type.
func FromAPIBillingSubscriptionCreateTiming(t apiv3models.BillingSubscriptionCreateTiming) (subscription.Timing, error) {
	// Try decoding as a custom RFC3339 datetime first, otherwise it would also decode
	// as a "string enum" and we'd never be able to distinguish enum vs datetime.
	if custom, err := t.AsDateTime(); err == nil {
		return subscription.Timing{Custom: &custom}, nil
	}

	enum, err := t.AsBillingSubscriptionCreateTimingEnum()
	if err != nil {
		return subscription.Timing{}, models.NewGenericValidationError(fmt.Errorf("invalid timing"))
	}

	if enum != apiv3models.BillingSubscriptionCreateTimingEnumImmediate {
		return subscription.Timing{}, models.NewGenericValidationError(fmt.Errorf("invalid timing: %s", enum))
	}

	return subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}, nil
}

// FromAPIBillingSubscriptionCreate converts a create subscription request to a create subscription workflow input.
func FromAPIBillingSubscriptionCreate(
	namespace string,
	customerID customer.CustomerID,
	subscriptionName string,
	createSubscriptionRequest apiv3models.BillingSubscriptionCreate,
) (subscriptionworkflow.CreateSubscriptionWorkflowInput, error) {
	metadata, err := labels.ToMetadata(createSubscriptionRequest.Labels)
	if err != nil {
		return subscriptionworkflow.CreateSubscriptionWorkflowInput{}, err
	}

	// Timing defaults to immediate when omitted; a future timing schedules the
	// subscription to start later. The domain validates the resolved timing
	// (rejecting a past instant), mirroring edit/change.
	timing := subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}
	if createSubscriptionRequest.Timing != nil {
		timing, err = FromAPIBillingSubscriptionCreateTiming(*createSubscriptionRequest.Timing)
		if err != nil {
			return subscriptionworkflow.CreateSubscriptionWorkflowInput{}, err
		}
	}

	workflowInput := subscriptionworkflow.CreateSubscriptionWorkflowInput{
		Namespace:       namespace,
		CustomerID:      customerID.ID,
		BillingAnchor:   createSubscriptionRequest.BillingAnchor,
		BillingStartsAt: createSubscriptionRequest.BillingStartsAt,
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Name:          subscriptionName,
			CostBasisMode: subscription.CostBasisMode(lo.FromPtr(createSubscriptionRequest.CostBasisMode)),
			Timing:        timing,
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
func FromAPIBillingSubscriptionEditOperation(op apiv3models.BillingSubscriptionEditOperation) (subscription.Patch, error) {
	disc, err := op.Discriminator()
	if err != nil {
		return nil, models.NewGenericValidationError(fmt.Errorf("failed to read edit operation type: %w", err))
	}

	switch disc {
	case string(apiv3models.BillingSubscriptionEditAddItemTypeAddItem):
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

	case string(apiv3models.BillingSubscriptionEditRemoveItemTypeRemoveItem):
		apiP, err := op.AsBillingSubscriptionEditRemoveItem()
		if err != nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("failed to decode remove_item operation: %w", err))
		}

		return patch.PatchRemoveItem{
			PhaseKey: apiP.PhaseKey,
			ItemKey:  apiP.ItemKey,
		}, nil

	case string(apiv3models.BillingSubscriptionEditAddPhaseTypeAddPhase):
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

	case string(apiv3models.BillingSubscriptionEditRemovePhaseTypeRemovePhase):
		apiP, err := op.AsBillingSubscriptionEditRemovePhase()
		if err != nil {
			return nil, models.NewGenericValidationError(fmt.Errorf("failed to decode remove_phase operation: %w", err))
		}

		var shift subscription.RemoveSubscriptionPhaseShifting
		switch apiP.Shift {
		case apiv3models.BillingSubscriptionRemovePhaseShiftingNext:
			shift = subscription.RemoveSubscriptionPhaseShiftNext
		case apiv3models.BillingSubscriptionRemovePhaseShiftingPrev:
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

	case string(apiv3models.BillingSubscriptionEditStretchPhaseTypeStretchPhase):
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

	case string(apiv3models.BillingSubscriptionEditUnscheduleEditTypeUnscheduleEdit):
		return patch.PatchUnscheduleEdit{}, nil

	default:
		return nil, models.NewGenericValidationError(fmt.Errorf("unknown edit operation type: %s", disc))
	}
}

// FromAPIBillingSubscriptionCustomPlan maps an inline (custom) plan definition from a
// create or change request into the domain plan create input. It performs pure
// mapping only: plan validity, currency resolution, and the fiat/custom-currency
// rules are enforced downstream by the subscription service and spec validation.
func FromAPIBillingSubscriptionCustomPlan(namespace string, body apiv3models.BillingSubscriptionCustomPlan) (plan.CreatePlanInput, error) {
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

func FromAPIBillingSubscriptionMigrate(id models.NamespacedID, body apiv3models.BillingSubscriptionMigrate) (plansubscription.MigrateSubscriptionRequest, error) {
	result := plansubscription.MigrateSubscriptionRequest{
		ID:            id,
		TargetVersion: body.TargetVersion,
		StartingPhase: body.StartingPhase,
		BillingAnchor: body.BillingAnchor,
	}

	// Preserve omission so the domain service can choose its default timing.
	if body.Timing != nil {
		timing, err := FromAPIBillingSubscriptionEditTiming(*body.Timing)
		if err != nil {
			return plansubscription.MigrateSubscriptionRequest{}, err
		}

		result.Timing = &timing
	}

	return result, nil
}
