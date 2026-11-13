package productcatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

// missingTaxCodeIDs returns the ids that don't reference a live Tax Code in the namespace. Every
// Tax Code reference check goes through here, so changing how soft-deleted codes are treated only
// touches this function.
//
// BatchResolve returns deleted codes too (old rows may still point at them), which is why the
// deleted check lives here rather than in the resolver.
func missingTaxCodeIDs(ctx context.Context, resolver TaxCodeResolver, namespace string, ids ...string) (map[string]struct{}, error) {
	ids = lo.Uniq(lo.Compact(ids))
	if len(ids) == 0 {
		return nil, nil
	}

	taxCodes, err := resolver.BatchResolve(ctx, namespace, ids...)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	missing := make(map[string]struct{})

	for _, id := range ids {
		if tc, ok := taxCodes[id]; !ok || tc == nil || tc.IsDeleted() {
			missing[id] = struct{}{}
		}
	}

	return missing, nil
}

// ValidateRateCardTaxCodes checks that every Tax Code the rate cards reference exists in the
// namespace and isn't deleted. Plan, add-on and subscription services all check Tax Code
// references here.
func ValidateRateCardTaxCodes(ctx context.Context, resolver TaxCodeResolver, namespace string, rateCards RateCards) error {
	missing, err := missingTaxCodeIDs(ctx, resolver, namespace, rateCards.TaxCodeIDs()...)
	if err != nil {
		return err
	}

	if len(missing) == 0 {
		return nil
	}

	var errs []error

	for _, rc := range rateCards {
		cfg := rc.AsMeta().TaxConfig
		if cfg == nil || cfg.TaxCodeID == "" {
			continue
		}

		if _, ok := missing[cfg.TaxCodeID]; !ok {
			continue
		}

		fieldSelector := models.NewFieldSelectorGroup(
			models.NewFieldSelector("ratecards").WithExpression(
				models.NewFieldAttrValue("key", rc.Key())),
		)

		errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector,
			fmt.Errorf("tax code not found [ratecard.key=%s tax_code.id=%s]: %w",
				rc.Key(), cfg.TaxCodeID, ErrRateCardTaxCodeNotFound),
		))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ValidateTaxCodeID checks that a single Tax Code reference exists in the namespace and isn't
// deleted, with the same rule as ValidateRateCardTaxCodes.
func ValidateTaxCodeID(ctx context.Context, resolver TaxCodeResolver, namespace string, id string) error {
	missing, err := missingTaxCodeIDs(ctx, resolver, namespace, id)
	if err != nil {
		return err
	}

	if _, ok := missing[id]; ok {
		return models.NewGenericValidationError(fmt.Errorf("tax code %s not found", id))
	}

	return nil
}
