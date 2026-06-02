package featureresolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
)

func ResolveFeaturesForRateCards(
	ctx context.Context,
	resolver productcatalog.FeatureResolver,
	namespace string,
	rateCards *productcatalog.RateCards,
) error {
	if rateCards == nil || len(*rateCards) == 0 {
		return nil
	}

	featureIDAndKeys := make([]string, 0, 2*len(*rateCards))

	for _, rc := range *rateCards {
		if !rc.HasFeature() {
			continue
		}

		if id := rc.GetFeatureID(); id != nil && *id != "" {
			featureIDAndKeys = append(featureIDAndKeys, *id)
		}

		if key := rc.GetFeatureKey(); key != nil && *key != "" {
			featureIDAndKeys = append(featureIDAndKeys, *key)
		}
	}

	features, err := resolver.BatchResolve(ctx, namespace, featureIDAndKeys...)
	if err != nil {
		return fmt.Errorf("failed to resolve features: %w", err)
	}

	var errs []error

	for _, rc := range *rateCards {
		if !rc.HasFeature() {
			continue
		}

		var f *feature.Feature

		id := rc.GetFeatureID()
		hasID := id != nil && *id != ""

		key := rc.GetFeatureKey()
		hasKey := key != nil && *key != ""

		fieldSelector := models.NewFieldSelectorGroup(
			models.NewFieldSelector("ratecards").WithExpression(
				models.NewFieldAttrValue("key", rc.Key())),
		)

		if hasID {
			f = features[*id]

			if f == nil {
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector,
					fmt.Errorf("feature not found [ratecard.key=%s feature.id=%s]: %w",
						rc.Key(), lo.FromPtr(id), productcatalog.ErrRateCardFeatureNotFound),
				))

				continue
			}

			if f.ID != *id {
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector,
					fmt.Errorf("feature id conflict [ratecard.key=%s feature.id=%s feature.key=%s]: %w",
						rc.Key(), lo.FromPtr(id), lo.FromPtr(key), productcatalog.ErrRateCardFeatureMismatch),
				))

				continue
			}
		}

		if hasKey {
			if f == nil {
				f = features[*key]
			}

			if f == nil {
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector,
					fmt.Errorf("feature not found [ratecard.key=%s feature.key=%s]: %w",
						rc.Key(), lo.FromPtr(key), productcatalog.ErrRateCardFeatureNotFound),
				))

				continue
			}

			if f.Key != *key {
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector,
					fmt.Errorf("feature key conflict [ratecard.key=%s feature.id=%s feature.key=%s]: %w",
						rc.Key(), lo.FromPtr(id), lo.FromPtr(key), productcatalog.ErrRateCardFeatureMismatch),
				))

				continue
			}
		}

		rc.SetFeature(&(f).ID, &(f).Key)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
