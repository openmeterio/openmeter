package patch

import (
	"maps"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

// patchItemSchedule replaces item versions from the first change onward.
// It handles scheduled addon quantity changes that add/remove-item cannot.
// Only migration uses this patch; it is not part of the public edit API.
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
	// Keep earlier and unchanged versions at their existing indexes.
	items := p.preservedPrefix(phase.ItemsByKey[p.itemKey], cadence)
	items = append(items, p.replacementSuffix(cadence)...)
	if len(items) == 0 {
		delete(phase.ItemsByKey, p.itemKey)
	} else {
		phase.ItemsByKey[p.itemKey] = items
	}
	return nil
}

// preservedPrefix keeps earlier versions and ends the active version at the
// first change. Their indexes stay the same so billing can match them.
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

// replacementSuffix adds the target versions from the first change onward.
// It keeps their scheduled start and end times, including gaps between versions.
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
