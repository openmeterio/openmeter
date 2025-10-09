package subscription

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func ValidateUniqueConstraintBySubscriptions(subs []SubscriptionSpec) error {
	var errs []error

	if overlaps := models.NewSortedCadenceList(subs).GetOverlaps(); len(overlaps) > 0 {
		for _, overlap := range overlaps {
			// to get proper selectors we'll add two errors (one for each side)
			errs = append(errs,
				ErrOnlySingleSubscriptionAllowed.WithAttrs(models.Attributes{
					ErrCodeOnlySingleSubscriptionAllowed: SubscriptionSubscriptionLevelUniqueConstraintErrorDetail{
						This: SubscriptionSubscriptionLevelUniqueConstraintErrorDetailSide{
							Subscription: overlap.Item1,
							Cadence:      overlap.Item1.GetCadence(),
							Selectors:    subscriptionSpecToFieldSelectors(overlap.Item1),
						},
						Other: SubscriptionSubscriptionLevelUniqueConstraintErrorDetailSide{
							Subscription: overlap.Item2,
							Cadence:      overlap.Item2.GetCadence(),
							Selectors:    subscriptionSpecToFieldSelectors(overlap.Item2),
						},
					},
				}).WithField(subscriptionSpecToFieldSelectors(overlap.Item1)))

			errs = append(errs,
				ErrOnlySingleSubscriptionAllowed.WithAttrs(models.Attributes{
					ErrCodeOnlySingleSubscriptionAllowed: SubscriptionSubscriptionLevelUniqueConstraintErrorDetail{
						This: SubscriptionSubscriptionLevelUniqueConstraintErrorDetailSide{
							Subscription: overlap.Item2,
							Cadence:      overlap.Item2.GetCadence(),
							Selectors:    subscriptionSpecToFieldSelectors(overlap.Item2),
						},
						Other: SubscriptionSubscriptionLevelUniqueConstraintErrorDetailSide{
							Subscription: overlap.Item1,
							Cadence:      overlap.Item1.GetCadence(),
							Selectors:    subscriptionSpecToFieldSelectors(overlap.Item1),
						},
					},
				}).WithField(subscriptionSpecToFieldSelectors(overlap.Item2)))
		}
	}

	return errors.Join(errs...)
}

func ValidateUniqueConstraintByFeatures(subs []SubscriptionSpec) error {
	return featureLevelUniqueConstraintValidator{}.Validate(subs)
}

type SubscriptionSubscriptionLevelUniqueConstraintErrorDetailSide struct {
	Subscription SubscriptionSpec        `json:"subscription"`
	Cadence      models.CadencedModel    `json:"cadence"`
	Selectors    *models.FieldDescriptor `json:"selectors"`
}

type SubscriptionSubscriptionLevelUniqueConstraintErrorDetail = models.Overlap[SubscriptionSubscriptionLevelUniqueConstraintErrorDetailSide]

type SubscriptionFeatureLevelUniqueConstraintErrorDetailSide struct {
	Item      SubscriptionItemSpec    `json:"-"` // useful internally but let's not expose it to the client
	Cadence   models.CadencedModel    `json:"cadence"`
	Selectors *models.FieldDescriptor `json:"selectors"`
	PlanRef   PlanRef                 `json:"plan_ref"`
}

type SubscriptionFeatureLevelUniqueConstraintErrorDetail = models.Overlap[SubscriptionFeatureLevelUniqueConstraintErrorDetailSide]

// let's localize all logic on this struct to avoid scope pollution
type featureLevelUniqueConstraintValidator struct{}

func (v featureLevelUniqueConstraintValidator) Validate(subs []SubscriptionSpec) error {
	relevantItems := v.collectRelevantItems(subs, v.itemIsRelevant)
	timelinesForRelevantItems, err := v.buildRelevantTimelines(relevantItems)
	if err != nil {
		return err
	}

	var errs []error
	for _, timeline := range timelinesForRelevantItems {
		if overlaps := timeline.GetOverlaps(); len(overlaps) > 0 {
			for _, overlap := range overlaps {
				// To get proper FieldSelectors, we'll add two errors (one for each side)
				errs = append(errs,
					ErrOnlySingleSubscriptionItemAllowedAtATime.
						WithAttrs(overlap.Item1.GetErrorAttributes(overlap.Item2)).
						WithField(overlap.Item1.Item.GetSelectors()))

				errs = append(errs,
					ErrOnlySingleSubscriptionItemAllowedAtATime.
						WithAttrs(overlap.Item2.GetErrorAttributes(overlap.Item1)).
						WithField(overlap.Item2.Item.GetSelectors()))
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

	for si := range subs {
		sub := subs[si]
		for pi := range sub.Phases {
			phase := sub.Phases[pi]
			for itemKey, items := range phase.ItemsByKey {
				for idx := range items {
					item := items[idx]
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

func (v featureLevelUniqueConstraintValidator) itemIsRelevant(item *SubscriptionItemSpec) bool {
	if item == nil {
		return false
	}

	return v.itemHasEntitlements(item) || v.itemIsBillable(item)
}

func (v featureLevelUniqueConstraintValidator) itemHasEntitlements(item *SubscriptionItemSpec) bool {
	if item == nil {
		return false
	}

	return item.RateCard.AsMeta().EntitlementTemplate != nil
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

func (i itemSpecWithCircularReferences) GetSelectors() *models.FieldDescriptor {
	return models.NewFieldSelectorGroup(
		subscriptionSpecToFieldSelectors(lo.FromPtr(i.SubscriptionSpec)),
		models.NewFieldSelector("phases").WithExpression(models.NewFieldAttrValue("key", i.SubscriptionPhaseSpec.PhaseKey)),
		models.NewFieldSelector("items").WithExpression(models.NewFieldAttrValue("key", i.SubscriptionItemSpec.ItemKey)),
		models.NewFieldSelector("idx").WithExpression(models.NewFieldArrIndex(i.SubscriptionItemVersion)),
	)
}

type validationTimelineEntry struct {
	Item    *itemSpecWithCircularReferences
	Cadence models.CadencedModel
}

func (i validationTimelineEntry) GetCadence() models.CadencedModel {
	return i.Cadence
}

func (i validationTimelineEntry) GetErrorAttributes(other validationTimelineEntry) models.Attributes {
	return models.Attributes{
		ErrCodeOnlySingleSubscriptionItemAllowedAtATime: SubscriptionFeatureLevelUniqueConstraintErrorDetail{
			This: SubscriptionFeatureLevelUniqueConstraintErrorDetailSide{
				Item:      lo.FromPtr(i.Item.SubscriptionItemSpec),
				PlanRef:   lo.FromPtr(i.Item.SubscriptionSpec.Plan),
				Cadence:   i.GetCadence(),
				Selectors: i.Item.GetSelectors(),
			},
			Other: SubscriptionFeatureLevelUniqueConstraintErrorDetailSide{
				Item:      lo.FromPtr(other.Item.SubscriptionItemSpec),
				PlanRef:   lo.FromPtr(other.Item.SubscriptionSpec.Plan),
				Cadence:   other.GetCadence(),
				Selectors: other.Item.GetSelectors(),
			},
		},
	}
}

func subscriptionSpecToFieldSelectors(subscriptionSpec SubscriptionSpec) *models.FieldDescriptor {
	selectors := []*models.FieldDescriptor{}

	if subscriptionSpec.Plan != nil {
		selectors = append(selectors, planRefToFieldSelector(subscriptionSpec.Plan))
	} else {
		selectors = append(selectors, models.NewFieldSelector("plans"))
	}

	selectors = append(selectors,
		models.NewFieldSelector("subscriptions").WithExpression(models.NewMultiFieldAttrValue(
			func() []models.FieldAttrValue {
				res := []models.FieldAttrValue{
					models.NewFieldAttrValue("customerId", subscriptionSpec.CustomerId),
					models.NewFieldAttrValue("activeFrom", subscriptionSpec.ActiveFrom),
				}

				if subscriptionSpec.ActiveTo != nil {
					res = append(res, models.NewFieldAttrValue("activeTo", subscriptionSpec.ActiveTo))
				}

				return res
			}()...,
		)))

	return models.NewFieldSelectorGroup(selectors...)
}

func planRefToFieldSelector(planRef *PlanRef) *models.FieldDescriptor {
	if planRef == nil {
		return models.NewFieldSelector("plans")
	}

	return models.NewFieldSelector("plans").WithExpression(models.NewMultiFieldAttrValue(
		func() []models.FieldAttrValue {
			res := []models.FieldAttrValue{}

			if planRef.Key != "" {
				res = append(res, models.NewFieldAttrValue("key", planRef.Key))
			}

			if planRef.Version != 0 {
				res = append(res, models.NewFieldAttrValue("version", planRef.Version))
			}

			if planRef.Id != "" {
				res = append(res, models.NewFieldAttrValue("id", planRef.Id))
			}

			return res
		}()...,
	))
}
