package subscription

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func ValidateUniqueConstraintBySubscriptions(subs []SubscriptionSpec) error {
	if overlaps := models.NewSortedCadenceList(subs).GetOverlaps(); len(overlaps) > 0 {
		return ErrOnlySingleSubscriptionAllowed.WithAttrs(models.Attributes{
			// FIXME[galexi]: improve on this
			"overlaps": overlaps,
		})
	}

	return nil
}

func ValidateUniqueConstraintByFeatures(subs []SubscriptionSpec) error {
	return featureLevelUniqueConstraintValidator{}.Validate(subs)
}

type SubscriptionUniqueConstraintErrorDetailSide struct {
	Item    SubscriptionItemSpec `json:"-"` // useful internally but let's not expose it to the client
	Path    SpecPath             `json:"path"`
	PlanRef PlanRef              `json:"plan_ref"`
}

type SubscriptionUniqueConstraintErrorDetail struct {
	Left  SubscriptionUniqueConstraintErrorDetailSide `json:"left"`
	Right SubscriptionUniqueConstraintErrorDetailSide `json:"right"`
}

// let's localize all logic on this struct to avoid scope pollution
type featureLevelUniqueConstraintValidator struct{}

func (v featureLevelUniqueConstraintValidator) Validate(subs []SubscriptionSpec) error {
	billableItems := v.collectRelevantItems(subs, v.itemIsBillable)
	timelinesForBillableItems, err := v.buildRelevantTimelines(billableItems)
	if err != nil {
		return err
	}

	var errs []error
	for _, timeline := range timelinesForBillableItems {
		if overlaps := timeline.GetOverlaps(); len(overlaps) > 0 {
			for _, overlap := range overlaps {
				errs = append(errs, ErrOnlySingleBillableItemAllowedAtATime.WithAttrs(models.Attributes{
					ErrCodeOnlySingleBillableItemAllowedAtATime: SubscriptionUniqueConstraintErrorDetail{
						Left: SubscriptionUniqueConstraintErrorDetailSide{
							Item:    lo.FromPtr(overlap.Item1.Item.SubscriptionItemSpec),
							Path:    overlap.Item1.Item.GetPath(),
							PlanRef: lo.FromPtr(overlap.Item1.Item.SubscriptionSpec.Plan),
						},
						Right: SubscriptionUniqueConstraintErrorDetailSide{
							Item:    lo.FromPtr(overlap.Item2.Item.SubscriptionItemSpec),
							Path:    overlap.Item2.Item.GetPath(),
							PlanRef: lo.FromPtr(overlap.Item2.Item.SubscriptionSpec.Plan),
						},
					},
				}))
			}
		}
	}

	return errors.Join(errs...)
}

func (v featureLevelUniqueConstraintValidator) buildRelevantTimelines(itemMap map[string][]itemSpecWithCircularReferences) (map[string]models.CadenceList[validationTimelineEntry], error) {
	timelines := make(map[string]models.CadenceList[validationTimelineEntry])

	for itemKey, items := range itemMap {
		validationTimelineEntries, err := slicesx.MapWithErr(items, func(item itemSpecWithCircularReferences) (validationTimelineEntry, error) {
			phaseCadence, err := item.SubscriptionSpec.GetPhaseCadence(item.SubscriptionPhaseSpec.PhaseKey)
			if err != nil {
				return validationTimelineEntry{}, fmt.Errorf("failed to get phase cadence for item %s: %w", itemKey, err)
			}

			itemCadence := item.SubscriptionItemSpec.GetCadence(phaseCadence)

			return validationTimelineEntry{
				Item:    &item,
				Cadence: itemCadence,
			}, nil
		})
		if err != nil {
			return nil, err
		}

		timelines[itemKey] = models.NewSortedCadenceList(validationTimelineEntries)
	}

	return timelines, nil
}

func (v featureLevelUniqueConstraintValidator) collectRelevantItems(subs []SubscriptionSpec, condition func(item *SubscriptionItemSpec) bool) map[string][]itemSpecWithCircularReferences {
	relevantItems := make(map[string][]itemSpecWithCircularReferences)

	for _, sub := range subs {
		for _, phase := range sub.Phases {
			for itemKey, items := range phase.ItemsByKey {
				for idx, item := range items {
					if condition(item) {
						relevantItems[itemKey] = append(relevantItems[itemKey], itemSpecWithCircularReferences{
							SubscriptionItemVersion: idx,
							SubscriptionItemSpec:    item,
							SubscriptionPhaseSpec:   phase,
							SubscriptionSpec:        &sub,
						})
					}
				}
			}
		}
	}

	return relevantItems
}

func (v featureLevelUniqueConstraintValidator) itemIsBillable(item *SubscriptionItemSpec) bool {
	if item == nil {
		return false
	}

	return item.RateCard.AsMeta().IsBillable()
}

// This will be a circular structure so be careful when using it
type itemSpecWithCircularReferences struct {
	SubscriptionItemVersion int
	SubscriptionItemSpec    *SubscriptionItemSpec
	SubscriptionPhaseSpec   *SubscriptionPhaseSpec
	SubscriptionSpec        *SubscriptionSpec
}

func (i itemSpecWithCircularReferences) GetPath() SpecPath {
	return NewItemVersionPath(i.SubscriptionPhaseSpec.PhaseKey, i.SubscriptionItemSpec.ItemKey, i.SubscriptionItemVersion)
}

type validationTimelineEntry struct {
	Item    *itemSpecWithCircularReferences
	Cadence models.CadencedModel
}

func (i validationTimelineEntry) GetCadence() models.CadencedModel {
	return i.Cadence
}
