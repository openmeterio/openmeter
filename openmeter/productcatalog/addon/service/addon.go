package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s service) ListAddons(ctx context.Context, params addon.ListAddonsInput) (pagination.Result[addon.Addon], error) {
	fn := func(ctx context.Context) (pagination.Result[addon.Addon], error) {
		if err := params.Validate(); err != nil {
			return pagination.Result[addon.Addon]{}, fmt.Errorf("invalid list add-ons params: %w", err)
		}

		return s.adapter.ListAddons(ctx, params)
	}

	return fn(ctx)
}

// resolveFeatures resolves the FeatureKey and FeatureID references for each RateCard
// - If FeatureID is provided (but not FeatureKey), it will populate FeatureKey
// - If FeatureKey is provided (but not FeatureID), it will populate FeatureID
// - If both FeatureKey and FeatureID are provided, it will validate that the provided key matches the value in the DB
//
// FIXME: this is a bit brittle, if any type implementing productcatalog.RateCard is not a pointer...
func (s service) resolveFeatures(ctx context.Context, namespace string, rateCards *productcatalog.RateCards) error {
	if rateCards == nil || len(*rateCards) == 0 {
		return nil
	}
	rateCardFeatureKeysOrIDs := make([]string, 0)
	for _, rateCard := range *rateCards {
		fK := rateCard.AsMeta().FeatureKey
		fID := rateCard.AsMeta().FeatureID

		if fK != nil {
			rateCardFeatureKeysOrIDs = append(rateCardFeatureKeysOrIDs, *fK)
		}

		if fID != nil {
			rateCardFeatureKeysOrIDs = append(rateCardFeatureKeysOrIDs, *fID)
		}
	}

	if len(rateCardFeatureKeysOrIDs) == 0 {
		return nil
	}

	featureList, err := s.feature.ListFeatures(ctx, feature.ListFeaturesParams{
		IDsOrKeys: rateCardFeatureKeysOrIDs,
		Namespace: namespace,
		Page: pagination.Page{
			PageSize:   len(rateCardFeatureKeysOrIDs), // gte to what will be returned
			PageNumber: 1,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list Features for RateCards: %w", err)
	}

	// Let's make a clone of it
	rateCardsClone := rateCards.Clone()

	for _, rateCard := range rateCardsClone {
		fK := rateCard.AsMeta().FeatureKey
		fID := rateCard.AsMeta().FeatureID

		if fID == nil && fK == nil {
			// We don't need to do anything, no feature is provided
			continue
		}

		var (
			featureByID    feature.Feature
			featureByKey   feature.Feature
			featureByIDOk  bool
			featureByKeyOk bool
		)

		if fID != nil {
			featureByID, featureByIDOk = lo.Find(featureList.Items, func(feat feature.Feature) bool {
				return feat.ID == *fID
			})
		}

		if fK != nil {
			featureByKey, featureByKeyOk = lo.Find(featureList.Items, func(feat feature.Feature) bool {
				return feat.Key == *fK
			})
		}

		if fID != nil && fK != nil {
			// We need to check that the two match (ID takes precedence)
			if !featureByIDOk {
				return models.NewGenericNotFoundError(fmt.Errorf("feature with ID %s not found", *fID))
			}

			if featureByID.Key != *fK {
				return models.NewGenericNotFoundError(fmt.Errorf("feature with ID %s has key %s, but expected %s", *fID, featureByID.Key, *fK))
			}
		} else if fID != nil && fK == nil {
			// We need to populate FeatureKey
			if !featureByIDOk {
				return models.NewGenericNotFoundError(fmt.Errorf("feature with ID %s not found", *fID))
			}

			// FIXME: merging like this is a pain, we should just use pointers...
			mNew := rateCard.AsMeta()
			mNew.FeatureKey = lo.ToPtr(featureByID.Key)
			var rcNew productcatalog.RateCard

			switch rateCard.Type() {
			case productcatalog.FlatFeeRateCardType:
				rcNew = &productcatalog.FlatFeeRateCard{
					RateCardMeta:   mNew,
					BillingCadence: rateCard.GetBillingCadence(),
				}
			case productcatalog.UsageBasedRateCardType:
				bc := rateCard.GetBillingCadence()
				if bc == nil {
					return models.NewGenericValidationError(fmt.Errorf("BillingCadence is required for UsageBasedRateCard"))
				}

				rcNew = &productcatalog.UsageBasedRateCard{
					RateCardMeta:   mNew,
					BillingCadence: *bc,
				}
			default:
				return models.NewGenericValidationError(fmt.Errorf("unsupported RateCard type: %s", rateCard.Type()))
			}

			if err = rateCard.Merge(rcNew); err != nil {
				return fmt.Errorf("failed to merge RateCard: %w", err)
			}
		} else if fID == nil && fK != nil {
			// We need to populate FeatureID
			if !featureByKeyOk {
				return models.NewGenericNotFoundError(fmt.Errorf("feature with key %s not found", *fK))
			}

			// FIXME: merging like this is a pain, we should just use pointers...
			mNew := rateCard.AsMeta()
			mNew.FeatureID = lo.ToPtr(featureByKey.ID)
			var rcNew productcatalog.RateCard

			switch rateCard.Type() {
			case productcatalog.FlatFeeRateCardType:
				rcNew = &productcatalog.FlatFeeRateCard{
					RateCardMeta:   mNew,
					BillingCadence: rateCard.GetBillingCadence(),
				}
			case productcatalog.UsageBasedRateCardType:
				bc := rateCard.GetBillingCadence()
				if bc == nil {
					return fmt.Errorf("billing cadence is required for usage-based rate card")
				}

				rcNew = &productcatalog.UsageBasedRateCard{
					RateCardMeta:   mNew,
					BillingCadence: *bc,
				}
			default:
				return fmt.Errorf("unsupported RateCard type: %s", rateCard.Type())
			}

			if err = rateCard.Merge(rcNew); err != nil {
				return fmt.Errorf("failed to merge RateCard: %w", err)
			}
		}
	}

	*rateCards = rateCardsClone

	return nil
}

// addonVersions is a collection of add-ons versions (all of them have the same namespace key pair).
type addonVersions []addon.Addon

func (a addonVersions) Len() int {
	return len(a)
}

func (a addonVersions) Less(i, j int) bool {
	return a[i].Version < a[j].Version
}

func (a addonVersions) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Sort sorts the add-ons by their versions.
func (a addonVersions) Sort() {
	sort.Sort(a)
}

// Latest returns add-on with the latest version regardless of its deleted status.
func (a addonVersions) Latest() *addon.Addon {
	if len(a) == 0 {
		return nil
	}

	// Ensure the collection is sorted
	a.Sort()

	return &a[len(a)-1]
}

// HasDraft returns true if there is an active (non-deleted) add-on with draft status.
func (a addonVersions) HasDraft() bool {
	for _, aa := range a {
		if aa.DeletedAt == nil && aa.Status() == productcatalog.AddonStatusDraft {
			return true
		}
	}

	return false
}

func (s service) getAddonVersions(ctx context.Context, namespace, key string) (addonVersions, error) {
	versions, err := s.adapter.ListAddons(ctx, addon.ListAddonsInput{
		OrderBy:        addon.OrderByVersion,
		Order:          addon.OrderAsc,
		Namespaces:     []string{namespace},
		Keys:           []string{key},
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list versions of the add-on: %w", err)
	}

	return versions.Items, nil
}

func (s service) CreateAddon(ctx context.Context, params addon.CreateAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context) (*addon.Addon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid create add-on params: %w", err)
		}

		logger := s.logger.With(
			"operation", "create",
			"namespace", params.Namespace,
			"addon.key", params.Key,
		)

		// Check if there is already an Add-on with the same Key
		versions, err := s.getAddonVersions(ctx, params.Namespace, params.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to get add-on generation: %w", err)
		}

		// Return error if the add-on generation already has an active (non-deleted) add-on with draft status
		// as there can only be single draft add-on at a time.
		if versions.HasDraft() {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("only a single draft version is allowed for add-on"),
			)
		}

		// Override the version parameter with the next version calculated from the last available version.
		params.Version = lo.FromPtr(versions.Latest()).Version + 1

		logger.Debug("creating add-on")

		if len(params.RateCards) > 0 {
			if err = s.resolveFeatures(ctx, params.Namespace, &params.RateCards); err != nil {
				if models.IsGenericNotFoundError(err) {
					err = models.NewGenericValidationError(err)
				}

				return nil, fmt.Errorf("failed to expand features for ratecards in add-on: %w", err)
			}
		}

		aa, err := s.adapter.CreateAddon(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create add-on: %w", err)
		}

		logger.With("addon.id", aa.ID).Debug("add-on created")

		// Emit add-on created event
		event := addon.NewAddonCreateEvent(ctx, aa)
		if err = s.publisher.Publish(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to publish add-on created event: %w", err)
		}

		return aa, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) DeleteAddon(ctx context.Context, params addon.DeleteAddonInput) error {
	fn := func(ctx context.Context) (interface{}, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid delete add-on params: %w", err)
		}

		logger := s.logger.With(
			"operation", "delete",
			"namespace", params.Namespace,
			"addon.id", params.ID,
		)

		logger.Debug("deleting add-on")

		// Get the add-on to check if it can be deleted
		add, err := s.adapter.GetAddon(ctx, addon.GetAddonInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
			Expand: addon.ExpandFields{
				PlanAddons: true,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get add-on: %w", err)
		}

		if add.DeletedAt != nil && add.DeletedAt.Before(clock.Now()) {
			return nil, nil
		}

		if add.Plans == nil {
			return nil, fmt.Errorf("cannot check whether add-on has plans enabled as plans were not dfetched for add-on [namespace=%s id=%s key=%s]",
				add.Namespace, add.ID, add.Key)
		}

		if len(*add.Plans) > 0 {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("failed to delete add-on [namespace=%s id=%s key=%s]: add-on has active assignments", add.Namespace, add.ID, add.Key),
			)
		}

		// Run validations prior deleting add-on.
		if err = add.AsProductCatalogAddon().ValidateWith(
			productcatalog.ValidateAddonWithStatus(productcatalog.AddonStatusDraft, productcatalog.AddonStatusArchived),
		); err != nil {
			return nil, err
		}

		// Delete the add-on
		err = s.adapter.DeleteAddon(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to delete add-on: %w", err)
		}

		logger.Debug("add-on deleted")

		// Get the deleted add-on to emit the event
		add, err = s.adapter.GetAddon(ctx, addon.GetAddonInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get deleted add-on: %w", err)
		}

		// Emit add-on deleted event
		event := addon.NewAddonDeleteEvent(ctx, add)
		if err = s.publisher.Publish(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to publish add-on deleted event: %w", err)
		}

		return nil, nil
	}

	_, err := transaction.Run(ctx, s.adapter, fn)

	return err
}

func (s service) GetAddon(ctx context.Context, params addon.GetAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context) (*addon.Addon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid get add-on params: %w", err)
		}

		logger := s.logger.With(
			"operation", "get",
			"namespace", params.Namespace,
			"addon.id", params.ID,
			"addon.key", params.Key,
			"addon.version", params.Version,
		)

		logger.Debug("fetching add-on")

		aa, err := s.adapter.GetAddon(ctx, params)
		if err != nil {
			// FIXME: not found error
			return nil, fmt.Errorf("failed to get add-on: %w", err)
		}

		logger.Debug("add-on fetched")

		return aa, nil
	}

	return fn(ctx)
}

func (s service) UpdateAddon(ctx context.Context, params addon.UpdateAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context) (*addon.Addon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid update add-on params: %w", err)
		}

		logger := s.logger.With(
			"operation", "update",
			"namespace", params.Namespace,
			"addon.id", params.ID,
		)
		logger.Debug("updating add-on")

		if params.RateCards != nil && len(*params.RateCards) > 0 {
			if err := s.resolveFeatures(ctx, params.Namespace, params.RateCards); err != nil {
				if models.IsGenericNotFoundError(err) {
					err = models.NewGenericValidationError(err)
				}

				return nil, fmt.Errorf("failed to expand features for ratecards in add-on: %w", err)
			}
		}

		add, err := s.adapter.GetAddon(ctx, addon.GetAddonInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get add-on: %w", err)
		}

		// Run validations prior updating add-on.
		if err = add.AsProductCatalogAddon().ValidateWith(
			productcatalog.ValidateAddonWithStatus(productcatalog.AddonStatusDraft),
		); err != nil {
			return nil, err
		}

		logger.Debug("updating add-on")

		// NOTE(chrisgacsal): we only allow updating the state of the add-on via Publish/Archive,
		// therefore the EffectivePeriod attribute must be zeroed before updating the add-on.
		params.EffectivePeriod = productcatalog.EffectivePeriod{}

		add, err = s.adapter.UpdateAddon(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to udpate add-on: %w", err)
		}

		logger.Debug("add-on updated")

		// Emit add-on updated event
		event := addon.NewAddonUpdateEvent(ctx, add)
		if err = s.publisher.Publish(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to publish add-on updated event: %w", err)
		}

		return add, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) PublishAddon(ctx context.Context, params addon.PublishAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context) (*addon.Addon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid publish add-on params: %w", err)
		}

		logger := s.logger.With(
			"operation", "publish",
			"namespace", params.Namespace,
			"addon.id", params.ID,
		)

		logger.Debug("publishing add-on")

		add, err := s.adapter.GetAddon(ctx, addon.GetAddonInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get add-on: %w", err)
		}

		if add.DeletedAt != nil {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("cannot publish a deleted add-on"),
			)
		}

		pa := add.AsProductCatalogAddon()

		// Run validations prior publishing add-on.

		var errs []error

		if err = pa.Publishable(); err != nil {
			errs = append(errs, fmt.Errorf("invalid add-on [id=%s key=%s version=%d]: %w",
				add.ID, add.Key, add.Version, err),
			)
		}

		// Validate plan with features
		resolver := productcatalog.NewNamespacedFeatureResolver(s.feature, params.Namespace)

		if err = pa.ValidateWith(productcatalog.ValidateAddonWithFeatures(ctx, resolver)); err != nil {
			errs = append(errs, fmt.Errorf("invalid add-on [id=%s key=%s version=%d]: %w",
				add.ID, add.Key, add.Version, err),
			)
		}

		if err = errors.Join(errs...); err != nil {
			return nil, models.NewGenericValidationError(err)
		}

		// Find and archive add-on version with addon.AddonStatusActive if there is one. Only perform lookup if
		// the add-on to be published has higher version then 1 meaning that it has previous versions,
		// otherwise skip this step.
		if add.Version > 1 {
			activeAddon, err := s.adapter.GetAddon(ctx, addon.GetAddonInput{
				NamespacedID: models.NamespacedID{
					Namespace: params.Namespace,
				},
				Key: add.Key,
			})
			if err != nil {
				if !addon.IsNotFound(err) {
					return nil, fmt.Errorf("failed to get add-on with active status: %w", err)
				}
			}

			if activeAddon != nil && params.EffectiveFrom != nil {
				_, err = s.ArchiveAddon(ctx, addon.ArchiveAddonInput{
					NamespacedID: models.NamespacedID{
						Namespace: activeAddon.Namespace,
						ID:        activeAddon.ID,
					},
					EffectiveTo: lo.FromPtr(params.EffectiveFrom),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to archive add-on with active status: %w", err)
				}
			}
		}

		// Publish new add-on version

		input := addon.UpdateAddonInput{
			NamespacedID: params.NamespacedID,
		}

		if params.EffectiveFrom != nil {
			input.EffectiveFrom = lo.ToPtr(params.EffectiveFrom.UTC())
		}

		if params.EffectiveTo != nil {
			input.EffectiveTo = lo.ToPtr(params.EffectiveTo.UTC())
		}

		add, err = s.adapter.UpdateAddon(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to publish add-on: %w", err)
		}

		logger.Debug("add-on published")

		// Emit add-on published event
		event := addon.NewAddonPublishEvent(ctx, add)
		if err := s.publisher.Publish(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to publish add-on published event: %w", err)
		}

		return add, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) ArchiveAddon(ctx context.Context, params addon.ArchiveAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context) (*addon.Addon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid archive add-on params: %w", err)
		}

		logger := s.logger.With(
			"operation", "archive",
			"namespace", params.Namespace,
			"addon.id", params.ID,
		)

		logger.Debug("archiving add-on")

		add, err := s.adapter.GetAddon(ctx, addon.GetAddonInput{
			NamespacedID: models.NamespacedID{
				Namespace: params.Namespace,
				ID:        params.ID,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get add-on: %w", err)
		}

		if add.DeletedAt != nil {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("cannot archive a deleted add-on"),
			)
		}

		// Run validations prior archiving add-on.
		if err = add.AsProductCatalogAddon().ValidateWith(
			productcatalog.ValidateAddonWithStatus(productcatalog.AddonStatusActive),
		); err != nil {
			return nil, err
		}

		add, err = s.adapter.UpdateAddon(ctx, addon.UpdateAddonInput{
			NamespacedID: models.NamespacedID{
				Namespace: add.Namespace,
				ID:        add.ID,
			},
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: add.EffectiveFrom,
				EffectiveTo:   lo.ToPtr(params.EffectiveTo.UTC()),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to archive add-on: %w", err)
		}

		logger.Debug("add-on archived")

		// Emit add-on archived event
		event := addon.NewAddonArchiveEvent(ctx, add)
		if err := s.publisher.Publish(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to publish add-on archived event: %w", err)
		}

		return add, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}

func (s service) NextAddon(ctx context.Context, params addon.NextAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context) (*addon.Addon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid next version add-on params: %w", err)
		}

		logger := s.logger.With(
			"operation", "next",
			"namespace", params.Namespace,
			"addon.id", params.ID,
			"addon.key", params.Key,
			"addon.version", params.Version,
		)

		logger.Debug("creating new version of an add-on")

		// Fetch all version of an add-on to find the one to be used as source and also to calculate the next version number.
		versions, err := s.getAddonVersions(ctx, params.Namespace, params.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to get add-on generation: %w", err)
		}

		if versions.Len() == 0 {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("no versions available for this add-on"),
			)
		}

		// Generate source add-on filter from input parameters

		// addonFilterFunc is a filter function which returns tuple where the first boolean means that
		// there is a match while the second tells the caller to stop further invocations as there is an exact match.
		type addonFilterFunc func(addon addon.Addon) (match bool, stop bool)

		sourceAddonFilterFunc := func() addonFilterFunc {
			switch {
			case params.ID != "":
				return func(a addon.Addon) (match bool, stop bool) {
					if a.Namespace == params.Namespace && a.ID == params.ID {
						return true, true
					}

					return false, false
				}
			case params.Key != "" && params.Version == 0:
				return func(a addon.Addon) (match bool, stop bool) {
					return a.Namespace == params.Namespace && a.Key == params.Key, false
				}
			default:
				return func(a addon.Addon) (match bool, stop bool) {
					if a.Namespace == params.Namespace && a.Key == params.Key && a.Version == params.Version {
						return true, true
					}

					return false, false
				}
			}
		}()

		var sourceAddon *addon.Addon

		nextVersion := 1
		var match, stop bool
		for _, addonItem := range versions {
			if addonItem.DeletedAt == nil && addonItem.Status() == productcatalog.AddonStatusDraft {
				return nil, models.NewGenericValidationError(
					fmt.Errorf("only a single draft version is allowed for add-on"),
				)
			}

			if !stop {
				match, stop = sourceAddonFilterFunc(addonItem)
				if match {
					sourceAddon = &addonItem
				}
			}

			if addonItem.Version >= nextVersion {
				nextVersion = addonItem.Version + 1
			}
		}

		if sourceAddon == nil {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("no versions available for add-on to use as source for next draft version"),
			)
		}

		nextAddon, err := s.adapter.CreateAddon(ctx, addon.CreateAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: sourceAddon.Namespace,
			},
			Addon: productcatalog.Addon{
				AddonMeta: productcatalog.AddonMeta{
					Key:         sourceAddon.Key,
					Version:     nextVersion,
					Name:        sourceAddon.Name,
					Description: sourceAddon.Description,
					Currency:    sourceAddon.Currency,
				},
				RateCards: sourceAddon.RateCards.AsProductCatalogRateCards(),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create new version of a add-on: %w", err)
		}

		return nextAddon, nil
	}

	return transaction.Run(ctx, s.adapter, fn)
}
