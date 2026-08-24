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
		reference := rc.AsMeta().Feature
		if reference == nil {
			continue
		}

		if id := reference.ID; id != nil && *id != "" {
			featureIDAndKeys = append(featureIDAndKeys, *id)
		}

		if key := reference.Key; key != nil && *key != "" {
			featureIDAndKeys = append(featureIDAndKeys, *key)
		}
	}

	features, err := resolver.BatchResolve(ctx, namespace, featureIDAndKeys...)
	if err != nil {
		return fmt.Errorf("failed to resolve features: %w", err)
	}

	var errs []error

	for _, rc := range *rateCards {
		fieldSelector := models.NewFieldSelectorGroup(
			models.NewFieldSelector("ratecards").WithExpression(
				models.NewFieldAttrValue("key", rc.Key())),
		)

		reference := rc.AsMeta().Feature
		if reference == nil {
			continue
		}

		if err := reference.Validate(); err != nil {
			errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, err))
			continue
		}

		var f *feature.Feature

		if reference.ID != nil {
			f = features[*reference.ID]

			if f == nil {
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector,
					fmt.Errorf("feature not found [ratecard.key=%s feature.id=%s]: %w",
						rc.Key(), lo.FromPtr(reference.ID), productcatalog.ErrRateCardFeatureNotFound),
				))

				continue
			}
		}

		if reference.Key != nil {
			if f == nil {
				f = features[*reference.Key]
			}

			if f == nil {
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector,
					fmt.Errorf("feature not found [ratecard.key=%s feature.key=%s]: %w",
						rc.Key(), lo.FromPtr(reference.Key), productcatalog.ErrRateCardFeatureNotFound),
				))

				continue
			}
		}

		if f == nil {
			errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector,
				fmt.Errorf("feature not found [ratecard.key=%s]: %w", rc.Key(), productcatalog.ErrRateCardFeatureNotFound),
			))
			continue
		}

		resolvedReference, err := reference.WithFeature(f)
		if err != nil {
			errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector,
				fmt.Errorf("feature reference conflict [ratecard.key=%s feature.id=%s feature.key=%s]: %w",
					rc.Key(), lo.FromPtr(reference.ID), lo.FromPtr(reference.Key), productcatalog.ErrRateCardFeatureMismatch),
			))
			continue
		}

		rc.SetFeatureReference(resolvedReference)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
