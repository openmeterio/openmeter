package patch

import (
	"maps"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

// patchItemSchedule replaces only the affected suffix. Unlike add/remove-item,
// it can address a schedule containing future addon quantity changes and gaps.
// It is internal to generated diffs, not an additional public edit operation.
type patchItemSchedule struct {
	phaseKey string
	itemKey  string
	at       time.Time
	target   []*subscription.SubscriptionItemSpec
}

func (p patchItemSchedule) Op() subscription.PatchOperation { return subscription.PatchOperationAdd }

func (p patchItemSchedule) Path() subscription.SpecPath {
	return subscription.NewItemPath(p.phaseKey, p.itemKey)
}
func (p patchItemSchedule) Validate() error { return p.Path().Validate() }

func (p patchItemSchedule) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
	if p.at.Before(actx.CurrentTime) {
		return &subscription.PatchForbiddenError{Msg: "cannot replace an item schedule in the past"}
	}
	phase, ok := spec.Phases[p.phaseKey]
	if !ok {
		return &subscription.PatchValidationError{Msg: "phase not found"}
	}
	cadence, err := spec.GetPhaseCadence(p.phaseKey)
	if err != nil {
		return err
	}
	// Preserve history and the unchanged prefix at their existing indexes.
	items := p.preservedPrefix(phase.ItemsByKey[p.itemKey], cadence)
	items = append(items, p.replacementSuffix(cadence)...)
	if len(items) == 0 {
		delete(phase.ItemsByKey, p.itemKey)
	} else {
		phase.ItemsByKey[p.itemKey] = items
	}
	return nil
}

// preservedPrefix closes a straddling item at the first difference and retains
// all earlier versions. Their positions are billing reconciliation identities.
func (p patchItemSchedule) preservedPrefix(current []*subscription.SubscriptionItemSpec, cadence models.CadencedModel) []*subscription.SubscriptionItemSpec {
	var items []*subscription.SubscriptionItemSpec
	for _, item := range current {
		c := item.GetCadence(cadence)
		if !c.ActiveFrom.Before(p.at) {
			break
		}
		kept := *item
		if c.ActiveTo == nil || c.ActiveTo.After(p.at) {
			kept.ActiveToOverrideRelativeToPhaseStart = lo.ToPtr(datetime.ISODurationBetween(cadence.ActiveFrom, p.at))
		}
		items = append(items, &kept)
	}
	return items
}

// replacementSuffix clips the target schedule to the amendment boundary while
// retaining its future quantity changes and gaps.
func (p patchItemSchedule) replacementSuffix(cadence models.CadencedModel) []*subscription.SubscriptionItemSpec {
	var items []*subscription.SubscriptionItemSpec
	for _, item := range p.target {
		c := item.GetCadence(cadence)
		if c.ActiveTo != nil && !c.ActiveTo.After(p.at) {
			continue
		}
		next := *item
		next.RateCard = item.RateCard.Clone()
		next.Annotations = maps.Clone(item.Annotations)
		if c.ActiveFrom.Before(p.at) {
			next.ActiveFromOverrideRelativeToPhaseStart = lo.ToPtr(datetime.ISODurationBetween(cadence.ActiveFrom, p.at))
		}
		items = append(items, &next)
	}
	return items
}
