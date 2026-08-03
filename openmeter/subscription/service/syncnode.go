package service

import (
	"context"
	"fmt"
	"maps"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type syncNodeKind string

const (
	syncNodeKindPhase       syncNodeKind = "phase"
	syncNodeKindEntitlement syncNodeKind = "entitlement"
	syncNodeKindItem        syncNodeKind = "item"
)

// syncNodeRef identifies one materialized resource. Items and entitlements
// intentionally share a spec path, so Kind distinguishes those two nodes.
type syncNodeRef struct {
	Path subscription.SpecPath
	Kind syncNodeKind
}

type syncNodeChange uint8

const (
	syncNodeChangeUnchanged syncNodeChange = iota
	syncNodeChangeCreate
	syncNodeChangeArchive
	syncNodeChangeReplace
)

// syncNode pairs the optional current and desired state of one materialized
// resource. Compare classifies the transition; Archive and Create apply it.
type syncNode interface {
	Ref() syncNodeRef
	Compare() (syncNodeChange, error)
	HasCurrent() bool
	HasDesired() bool
	forceChange() syncNodeChange
	Archive(context.Context, *syncExecution) error
	Create(context.Context, *syncExecution) error
}

type phaseSyncNode struct {
	ref            syncNodeRef
	subscription   subscription.Subscription
	current        *subscription.SubscriptionPhaseView
	desired        *subscription.SubscriptionPhaseSpec
	desiredCadence models.CadencedModel
}

func (n *phaseSyncNode) Ref() syncNodeRef {
	return n.ref
}

func (n *phaseSyncNode) HasCurrent() bool {
	return n.current != nil
}

func (n *phaseSyncNode) HasDesired() bool {
	return n.desired != nil
}

func (n *phaseSyncNode) forceChange() syncNodeChange {
	switch {
	case n.HasCurrent() && n.HasDesired():
		return syncNodeChangeReplace
	case n.HasCurrent():
		return syncNodeChangeArchive
	case n.HasDesired():
		return syncNodeChangeCreate
	default:
		return syncNodeChangeUnchanged
	}
}

func (n *phaseSyncNode) Compare() (syncNodeChange, error) {
	return compareSyncNodePresence(n.HasCurrent(), n.HasDesired(), func() (bool, error) {
		current := n.current.Spec.ToCreateSubscriptionPhaseEntityInput(
			n.subscription,
			n.current.SubscriptionPhase.ActiveFrom,
		)
		desired := n.desired.ToCreateSubscriptionPhaseEntityInput(
			n.subscription,
			n.desiredCadence.ActiveFrom,
		)

		// StartAfter is source-spec timing. The persisted phase materializes its
		// absolute ActiveFrom, and rebuilding a view may express the same offset
		// with a different but equivalent ISO duration.
		current.StartAfter = desired.StartAfter

		return current.Equal(desired), nil
	})
}

func (n *phaseSyncNode) Archive(ctx context.Context, execution *syncExecution) error {
	if n.current == nil {
		return fmt.Errorf("phase has no current state")
	}

	if err := execution.service.deletePhase(ctx, n.current.SubscriptionPhase); err != nil {
		return err
	}

	delete(execution.refState.phases, n.Ref().Path)
	return nil
}

func (n *phaseSyncNode) Create(ctx context.Context, execution *syncExecution) error {
	if n.desired == nil {
		return fmt.Errorf("phase has no desired state")
	}

	phase, err := execution.service.createPhase(ctx, *n.desired, n.subscription, n.desiredCadence)
	if err != nil {
		return err
	}

	execution.refState.phases[n.Ref().Path] = phase
	return nil
}

type itemSyncNode struct {
	ref                 syncNodeRef
	phasePath           subscription.SpecPath
	subscription        subscription.Subscription
	current             *subscription.SubscriptionItemView
	desired             *subscription.SubscriptionItemSpec
	currentPhaseCadence models.CadencedModel
	desiredPhaseCadence models.CadencedModel
}

func (n *itemSyncNode) Ref() syncNodeRef {
	return n.ref
}

func (n *itemSyncNode) HasCurrent() bool {
	return n.current != nil
}

func (n *itemSyncNode) HasDesired() bool {
	return n.desired != nil
}

func (n *itemSyncNode) forceChange() syncNodeChange {
	switch {
	case n.HasCurrent() && n.HasDesired():
		return syncNodeChangeReplace
	case n.HasCurrent():
		return syncNodeChangeArchive
	case n.HasDesired():
		return syncNodeChangeCreate
	default:
		return syncNodeChangeUnchanged
	}
}

func (n *itemSyncNode) Compare() (syncNodeChange, error) {
	return compareSyncNodePresence(n.HasCurrent(), n.HasDesired(), func() (bool, error) {
		comparisonPhaseID := models.NamespacedID{Namespace: n.subscription.Namespace}

		current, err := n.current.Spec.ToCreateSubscriptionItemEntityInput(
			comparisonPhaseID,
			n.currentPhaseCadence,
			nil,
		)
		if err != nil {
			return false, fmt.Errorf("failed to derive current item state: %w", err)
		}

		desired, err := n.desired.ToCreateSubscriptionItemEntityInput(
			comparisonPhaseID,
			n.desiredPhaseCadence,
			nil,
		)
		if err != nil {
			return false, fmt.Errorf("failed to derive desired item state: %w", err)
		}

		return current.Equal(desired), nil
	})
}

func (n *itemSyncNode) Archive(ctx context.Context, execution *syncExecution) error {
	if n.current == nil {
		return fmt.Errorf("item has no current state")
	}

	if err := execution.service.deleteItem(ctx, n.current.SubscriptionItem); err != nil {
		return err
	}

	return nil
}

func (n *itemSyncNode) Create(ctx context.Context, execution *syncExecution) error {
	if n.desired == nil {
		return fmt.Errorf("item has no desired state")
	}

	phase, ok := execution.refState.phases[n.phasePath]
	if !ok {
		return fmt.Errorf("phase reference %s is not materialized", n.phasePath)
	}

	entitlement := execution.refState.entitlements[n.Ref().Path]
	_, err := execution.service.createItem(ctx, createItemOptions{
		cust:         execution.customer,
		sub:          n.subscription,
		phase:        phase,
		phaseCadence: n.desiredPhaseCadence,
		itemSpec:     *n.desired,
	}, entitlement)
	if err != nil {
		return err
	}

	return nil
}

type entitlementSyncNode struct {
	ref          syncNodeRef
	subscription subscription.Subscription
	currentItem  *subscription.SubscriptionItemView
	current      *subscription.SubscriptionEntitlement
	desired      *subscription.ScheduleSubscriptionEntitlementInput
}

func (n *entitlementSyncNode) Ref() syncNodeRef {
	return n.ref
}

func (n *entitlementSyncNode) HasCurrent() bool {
	return n.current != nil
}

func (n *entitlementSyncNode) HasDesired() bool {
	return n.desired != nil
}

func (n *entitlementSyncNode) forceChange() syncNodeChange {
	switch {
	case n.HasCurrent() && n.HasDesired():
		return syncNodeChangeReplace
	case n.HasCurrent():
		return syncNodeChangeArchive
	case n.HasDesired():
		return syncNodeChangeCreate
	default:
		return syncNodeChangeUnchanged
	}
}

func (n *entitlementSyncNode) Compare() (syncNodeChange, error) {
	return compareSyncNodePresence(n.HasCurrent(), n.HasDesired(), func() (bool, error) {
		current := normalizeEntitlementInput(
			n.current.ToScheduleSubscriptionEntitlementInput(),
			n.subscription.ID,
		)
		desired := normalizeEntitlementInput(*n.desired, n.subscription.ID)

		return current.Equal(desired), nil
	})
}

func (n *entitlementSyncNode) Archive(ctx context.Context, execution *syncExecution) error {
	if n.current == nil || n.currentItem == nil {
		return fmt.Errorf("entitlement has no current item state")
	}

	if err := execution.service.deleteItemEntitlement(ctx, n.currentItem.SubscriptionItem); err != nil {
		return err
	}

	delete(execution.refState.entitlements, n.Ref().Path)
	return nil
}

func (n *entitlementSyncNode) Create(ctx context.Context, execution *syncExecution) error {
	if n.desired == nil {
		return fmt.Errorf("entitlement has no desired state")
	}

	entitlement, err := execution.service.createItemEntitlement(ctx, n.subscription, *n.desired)
	if err != nil {
		return err
	}

	execution.refState.entitlements[n.Ref().Path] = entitlement
	return nil
}

func normalizeEntitlementInput(
	input subscription.ScheduleSubscriptionEntitlementInput,
	subscriptionID string,
) subscription.ScheduleSubscriptionEntitlementInput {
	input.FeatureID = nil
	input.SubscriptionManaged = true
	input.Annotations = maps.Clone(input.Annotations)
	if input.Annotations == nil {
		input.Annotations = models.Annotations{}
	}
	input.Annotations[subscription.AnnotationSubscriptionID] = subscriptionID

	// Entitlement persistence stores the recurrence as effective from
	// MeasureUsageFrom, while the desired input carries its billing anchor.
	if input.UsagePeriod != nil && input.MeasureUsageFrom != nil {
		usagePeriod := entitlement.NewStartingUsagePeriodInput(
			input.UsagePeriod.GetValue(),
			input.MeasureUsageFrom.Get(),
		)
		input.UsagePeriod = &usagePeriod
	}

	return input
}

func compareSyncNodePresence(
	hasCurrent bool,
	hasDesired bool,
	equal func() (bool, error),
) (syncNodeChange, error) {
	switch {
	case !hasCurrent && hasDesired:
		return syncNodeChangeCreate, nil
	case hasCurrent && !hasDesired:
		return syncNodeChangeArchive, nil
	case !hasCurrent:
		return syncNodeChangeUnchanged, nil
	}

	isEqual, err := equal()
	if err != nil {
		return syncNodeChangeUnchanged, err
	}
	if isEqual {
		return syncNodeChangeUnchanged, nil
	}

	return syncNodeChangeReplace, nil
}
