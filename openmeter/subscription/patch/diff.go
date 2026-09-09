package patch

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

type DiffItemsInput struct {
	Current subscription.SubscriptionSpec
	Target  subscription.SubscriptionSpec
	At      time.Time
}

func (i DiffItemsInput) Validate() error {
	var errs []error
	if i.At.IsZero() {
		errs = append(errs, errors.New("effective time is required"))
	}
	if err := i.Current.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("current spec: %w", err))
	}
	if err := i.Target.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("target spec: %w", err))
	}
	if !(models.CadencedModel{ActiveFrom: i.Current.ActiveFrom, ActiveTo: i.Current.ActiveTo}).Equal(models.CadencedModel{
		ActiveFrom: i.Target.ActiveFrom, ActiveTo: i.Target.ActiveTo,
	}) || len(i.Current.Phases) != len(i.Target.Phases) {
		errs = append(errs, errors.New("item diff requires the same phase timeline"))
	}
	for key, phase := range i.Current.Phases {
		target, ok := i.Target.Phases[key]
		if !ok || phase == nil || target == nil {
			errs = append(errs, fmt.Errorf("item diff requires the same start for phase %q", key))
			continue
		}
		currentStart, _ := phase.StartAfter.AddTo(i.Current.ActiveFrom)
		targetStart, _ := target.StartAfter.AddTo(i.Target.ActiveFrom)
		if !currentStart.Equal(targetStart) {
			errs = append(errs, fmt.Errorf("item diff requires the same start for phase %q", key))
		}
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// DiffItems compares commercial schedules from At onward. Historical versions
// and the unchanged prefix of each item schedule retain their slice positions,
// which are the identities used by billing reconciliation.
func DiffItems(input DiffItemsInput) ([]subscription.Patch, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	var patches []subscription.Patch
	for _, phase := range input.Current.GetSortedPhases() {
		cadence, err := input.Current.GetPhaseCadence(phase.PhaseKey)
		if err != nil {
			return nil, err
		}
		if cadence.ActiveTo != nil && !cadence.ActiveTo.After(input.At) {
			continue
		}
		at := input.At
		if cadence.ActiveFrom.After(at) {
			at = cadence.ActiveFrom
		}
		targetPhase := input.Target.Phases[phase.PhaseKey]
		keys := slices.Collect(maps.Keys(phase.ItemsByKey))
		keys = append(keys, slices.Collect(maps.Keys(targetPhase.ItemsByKey))...)
		slices.Sort(keys)
		for _, key := range slices.Compact(keys) {
			current := phase.ItemsByKey[key]
			target := targetPhase.ItemsByKey[key]
			boundaries := []time.Time{at}
			for _, items := range [][]*subscription.SubscriptionItemSpec{current, target} {
				for _, item := range items {
					c := item.GetCadence(cadence)
					if c.ActiveFrom.After(at) {
						boundaries = append(boundaries, c.ActiveFrom)
					}
					if c.ActiveTo != nil && c.ActiveTo.After(at) {
						boundaries = append(boundaries, *c.ActiveTo)
					}
				}
			}
			slices.SortFunc(boundaries, time.Time.Compare)
			for _, boundary := range boundaries {
				if cadence.ActiveTo != nil && !boundary.Before(*cadence.ActiveTo) {
					break
				}
				if itemShapeEqual(itemAt(current, cadence, boundary), itemAt(target, cadence, boundary)) {
					continue
				}
				patches = append(patches, patchItemSchedule{phaseKey: phase.PhaseKey, itemKey: key, at: boundary, target: target})
				break
			}
		}
	}
	return patches, nil
}

func itemAt(items []*subscription.SubscriptionItemSpec, phase models.CadencedModel, at time.Time) *subscription.SubscriptionItemSpec {
	for _, item := range items {
		if item.GetCadence(phase).IsActiveAt(at) {
			return item
		}
	}
	return nil
}

func itemShapeEqual(a, b *subscription.SubscriptionItemSpec) bool {
	if a == nil || b == nil {
		return a == b
	}
	// Stored items can carry key-only feature references, while catalog rate cards
	// include both identifiers. Compare the shared identity without treating that
	// representational difference as an amendment.
	left, right := a.RateCard.Clone(), b.RateCard.Clone()
	lFeature, rFeature := left.AsMeta().Feature, right.AsMeta().Feature
	if lFeature != nil && rFeature != nil && lFeature.Compatible(*rFeature) &&
		((lFeature.Key != nil && rFeature.Key != nil) || (lFeature.ID != nil && rFeature.ID != nil)) {
		if err := right.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
			meta.Feature = lFeature
			return meta, nil
		}); err != nil {
			return false
		}
	}
	return left.Equal(right) && reflect.DeepEqual(a.BillingBehaviorOverride, b.BillingBehaviorOverride) &&
		isSubscriptionOwned(a.Annotations) == isSubscriptionOwned(b.Annotations) &&
		subscription.AnnotationParser.GetBooleanEntitlementCount(a.Annotations) == subscription.AnnotationParser.GetBooleanEntitlementCount(b.Annotations)
}

// Ownership determines whether addon removal may delete an otherwise empty item.
func isSubscriptionOwned(annotations models.Annotations) bool {
	if owners, ok := annotations[subscription.AnnotationOwnerSubSystem].([]string); ok {
		return slices.Contains(owners, subscription.OwnerSubscriptionSubSystem)
	}
	return slices.Contains(subscription.AnnotationParser.ListOwnerSubSystems(annotations), subscription.OwnerSubscriptionSubSystem)
}

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
	var items []*subscription.SubscriptionItemSpec
	for _, item := range phase.ItemsByKey[p.itemKey] {
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
	if len(items) == 0 {
		delete(phase.ItemsByKey, p.itemKey)
	} else {
		phase.ItemsByKey[p.itemKey] = items
	}
	return nil
}
