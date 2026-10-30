package productcatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ValidateRateCardTaxCodes checks that every Tax Code the rate cards reference exists in the
// namespace and isn't deleted. Plan, add-on and subscription services all check Tax Code
// references here, so changing how soft-deleted codes are treated only touches this function.
//
// BatchResolve returns deleted codes too (old rows may still point at them), which is why the
// deleted check lives here rather than in the resolver.
func ValidateRateCardTaxCodes(ctx context.Context, resolver TaxCodeResolver, namespace string, rateCards RateCards) error {
	ids := rateCards.TaxCodeIDs()
	if len(ids) == 0 {
		return nil
	}

	taxCodes, err := resolver.BatchResolve(ctx, namespace, ids...)
	if err != nil {
		return fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	var errs []error

	for _, rc := range rateCards {
		cfg := rc.AsMeta().TaxConfig
		if cfg == nil || cfg.TaxCodeID == "" {
			continue
		}

		if tc, ok := taxCodes[cfg.TaxCodeID]; ok && tc != nil && !tc.IsDeleted() {
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
