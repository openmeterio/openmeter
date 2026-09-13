package patch

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
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
	currentCadence := models.CadencedModel{ActiveFrom: i.Current.ActiveFrom, ActiveTo: i.Current.ActiveTo}
	targetCadence := models.CadencedModel{ActiveFrom: i.Target.ActiveFrom, ActiveTo: i.Target.ActiveTo}
	if !currentCadence.Equal(targetCadence) || len(i.Current.Phases) != len(i.Target.Phases) {
		errs = append(errs, errors.New("item diff requires the same phase timeline"))
	}
	for key, phase := range i.Current.Phases {
		target, ok := i.Target.Phases[key]
		if !ok || phase == nil || target == nil {
			errs = append(errs, fmt.Errorf("item diff requires phase %q in both specs", key))
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

// DiffItems compares item versions from At onward. It keeps earlier and
// unchanged versions at the same indexes because billing uses those indexes
// to match subscription items to invoice lines.
func DiffItems(input DiffItemsInput) ([]subscription.Patch, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	var patches []subscription.Patch
	for _, phase := range input.Current.GetSortedPhases() {
		// Skip past phases. Compare future phases from their start time.
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
		// Include keys from both plans to find added and removed items.
		keys := slices.Collect(maps.Keys(phase.ItemsByKey))
		keys = append(keys, slices.Collect(maps.Keys(targetPhase.ItemsByKey))...)
		slices.Sort(keys)
		for _, key := range slices.Compact(keys) {
			current := phase.ItemsByKey[key]
			target := targetPhase.ItemsByKey[key]
			diff := itemScheduleDiff{current: current, target: target, cadence: cadence, from: at}
			if boundary, changed := diff.firstDifference(); changed {
				patches = append(patches, patchItemSchedule{phaseKey: phase.PhaseKey, itemKey: key, at: boundary, target: target})
			}
		}
	}
	return patches, nil
}

// An item can differ only when one of its versions starts or ends. Check
// those times in both specs, including times when neither has an active item.
type itemScheduleDiff struct {
	current, target []*subscription.SubscriptionItemSpec
	cadence         models.CadencedModel
	from            time.Time
}

func (d itemScheduleDiff) firstDifference() (time.Time, bool) {
	for _, boundary := range d.boundaries() {
		if d.cadence.ActiveTo != nil && !boundary.Before(*d.cadence.ActiveTo) {
			break
		}
		current := itemAt(d.current, d.cadence, boundary)
		target := itemAt(d.target, d.cadence, boundary)
		if !itemShapeEqual(current, target) {
			return boundary, true
		}
	}
	return time.Time{}, false
}

func (d itemScheduleDiff) boundaries() []time.Time {
	boundaries := []time.Time{d.from}
	for _, items := range [][]*subscription.SubscriptionItemSpec{d.current, d.target} {
		for _, item := range items {
			cadence := item.GetCadence(d.cadence)
			if cadence.ActiveFrom.After(d.from) {
				boundaries = append(boundaries, cadence.ActiveFrom)
			}
			if cadence.ActiveTo != nil && cadence.ActiveTo.After(d.from) {
				boundaries = append(boundaries, *cadence.ActiveTo)
			}
		}
	}
	slices.SortFunc(boundaries, time.Time.Compare)
	return slices.CompactFunc(boundaries, time.Time.Equal)
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
	// Stored items may identify a feature by key while the plan also has its ID.
	// That alone does not mean the feature changed.
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
