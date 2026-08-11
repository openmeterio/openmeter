package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	customcurrencydb "github.com/openmeterio/openmeter/openmeter/ent/db/customcurrency"
	featuredb "github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	taxcodedb "github.com/openmeterio/openmeter/openmeter/ent/db/taxcode"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// validatePlanReferences temporarily enforces that all plan references belong to the same namespace.
func (a *adapter) validatePlanReferences(ctx context.Context, namespace string, p productcatalog.Plan) error {
	customCurrencyIDs := make([]string, 0)
	featureIDs := make([]string, 0)
	taxCodeIDs := make([]string, 0)

	if p.Currency.CustomCurrencyID != nil {
		customCurrencyIDs = append(customCurrencyIDs, *p.Currency.CustomCurrencyID)
	}

	for _, phase := range p.Phases {
		for _, rateCard := range phase.RateCards {
			meta := rateCard.AsMeta()

			if meta.Currency != nil && meta.Currency.CustomCurrencyID != nil {
				customCurrencyIDs = append(customCurrencyIDs, *meta.Currency.CustomCurrencyID)
			}
			if meta.FeatureID != nil {
				featureIDs = append(featureIDs, *meta.FeatureID)
			}
			if meta.TaxConfig != nil && meta.TaxConfig.TaxCodeID != nil {
				taxCodeIDs = append(taxCodeIDs, *meta.TaxConfig.TaxCodeID)
			}
		}
	}

	customCurrencyIDs = lo.Uniq(customCurrencyIDs)
	if len(customCurrencyIDs) > 0 {
		count, err := a.db.CustomCurrency.Query().
			Where(
				customcurrencydb.Namespace(namespace),
				customcurrencydb.IDIn(customCurrencyIDs...),
				customcurrencydb.DeletedAtIsNil(),
			).
			Count(ctx)
		if err != nil {
			return fmt.Errorf("validating plan custom currencies: %w", err)
		}
		if count != len(customCurrencyIDs) {
			return models.NewGenericValidationError(productcatalog.ErrCurrencyNotFound)
		}
	}

	featureIDs = lo.Uniq(featureIDs)
	if len(featureIDs) > 0 {
		count, err := a.db.Feature.Query().
			Where(
				featuredb.Namespace(namespace),
				featuredb.IDIn(featureIDs...),
				featuredb.ArchivedAtIsNil(),
			).
			Count(ctx)
		if err != nil {
			return fmt.Errorf("validating plan features: %w", err)
		}
		if count != len(featureIDs) {
			return models.NewGenericValidationError(productcatalog.ErrRateCardFeatureNotFound)
		}
	}

	taxCodeIDs = lo.Uniq(taxCodeIDs)
	if len(taxCodeIDs) > 0 {
		count, err := a.db.TaxCode.Query().
			Where(
				taxcodedb.Namespace(namespace),
				taxcodedb.IDIn(taxCodeIDs...),
				taxcodedb.DeletedAtIsNil(),
			).
			Count(ctx)
		if err != nil {
			return fmt.Errorf("validating plan tax codes: %w", err)
		}
		if count != len(taxCodeIDs) {
			return models.NewGenericValidationError(taxcode.ErrTaxCodeNotFound)
		}
	}

	return nil
}
