package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

// resolveLineTaxConfig resolves one line's v1 tax config inside the request's transaction,
// replacing the input's stripe.code with the referenced Tax Code's ID. The input's stripe.code
// is dropped: the stored config references the Tax Code, and responses render stripe.code from
// its current mapping. raw is the config as the request mapper produced it.
func resolveLineTaxConfig(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver, namespace string, raw *productcatalog.TaxConfig) (*productcatalog.TaxConfig, error) {
	if raw == nil {
		return nil, nil
	}

	// FromAPITaxConfig carries the stripe.code format check, the empty-ID rejection and
	// taxCodeId-wins; non-resolution errors keep the 400 class through the wrapper below.
	cfg, err := productcataloghttp.FromAPITaxConfig(ctx, resolver, namespace, productcataloghttp.FromTaxConfig(*raw))
	if err != nil {
		if isTaxCodeResolutionError(err) {
			return nil, err
		}

		return nil, billing.ValidationError{Err: err}
	}

	// cfg is nil only for a config with no behavior, ID or code; keep it as the request sent it.
	if cfg == nil {
		return raw, nil
	}

	out := cfg.ToTaxConfig()

	return &out, nil
}

// isTaxCodeResolutionError reports whether the error is a resolution failure (a 500), not bad
// request input.
func isTaxCodeResolutionError(err error) bool {
	var resolveErr *productcataloghttp.TaxCodeResolutionError
	return errors.As(err, &resolveErr)
}

// lineTaxConfig returns the raw tax config the request mappers use for a line: the deprecated
// top-level taxConfig when there is no rate card, the rate card's otherwise. Requests where the
// two differ are rejected (by the update mappers, and before resolution for simulation), so either
// read gives the same config.
func lineTaxConfig(rateCard *apiv1models.InvoiceUsageBasedRateCard, taxConfig *apiv1models.TaxConfig) *apiv1models.TaxConfig {
	if rateCard == nil {
		return taxConfig
	}

	return rateCard.TaxConfig
}

// resolveUpdateLineTaxConfigs resolves every edited line's tax config up front, so the update's
// EditFn (which has no context) can apply the already-resolved IDs. The result has one entry per
// request line, in the same order; an entry is nil when the line sent no tax config.
func resolveUpdateLineTaxConfigs(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver, namespace string, lines []apiv1models.InvoiceLineReplaceUpdate) ([]*productcatalog.TaxConfig, error) {
	resolved := make([]*productcatalog.TaxConfig, 0, len(lines))

	for _, line := range lines {
		raw := lineTaxConfig(line.RateCard, line.TaxConfig)
		if raw == nil {
			resolved = append(resolved, nil)

			continue
		}

		rawConfig := productcataloghttp.AsTaxConfig(*raw)
		cfg, err := resolveLineTaxConfig(ctx, resolver, namespace, &rawConfig)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, cfg)
	}

	return resolved, nil
}

// resolveSimulationLineTaxConfigs replaces each simulation line's stripe.code with the ID of the
// Tax Code it maps to. Simulation must not write, so the lookup is read-only: an unknown code
// leaves the line as sent, and the response shows it without stripe.code. A taxCodeId the
// request sent must name a live Tax Code, like on create and update; otherwise rendering the
// response would fail on it.
func resolveSimulationLineTaxConfigs(ctx context.Context, resolver productcatalog.TransactionalTaxCodeResolver, namespace string, lines []apiv1models.InvoiceSimulationLine) error {
	var requestRefs []billing.RequestTaxCodeRef

	for i := range lines {
		// Compare before resolving: resolution writes the same config to both fields, which
		// would hide a mismatch from the mapper's own check.
		if lines[i].TaxConfig != nil && lines[i].RateCard != nil && lines[i].RateCard.TaxConfig != nil &&
			!reflect.DeepEqual(*lines[i].TaxConfig, *lines[i].RateCard.TaxConfig) {
			return billing.ValidationError{Err: fmt.Errorf("lines/%d: %w", i, errTaxConfigMismatch)}
		}

		raw := lineTaxConfig(lines[i].RateCard, lines[i].TaxConfig)
		if raw == nil {
			continue
		}

		cfg, stripeCode, err := productcataloghttp.ParseAPITaxConfig(*raw)
		if err != nil {
			return billing.ValidationError{Err: err}
		}

		// ParseAPITaxConfig leaves the stripe code empty when the request sent a taxCodeId, so
		// an ID here came from the request and not from the stripe lookup below.
		if cfg.TaxCodeID != "" {
			requestRefs = append(requestRefs, billing.RequestTaxCodeRef{
				TaxCodeID: cfg.TaxCodeID,
				FieldPath: fmt.Sprintf("lines/%d/tax_config/tax_code", i),
			})
		}

		if stripeCode != "" {
			tc, err := resolver.GetByStripeCode(ctx, productcatalog.GetTaxCodeByStripeCodeInput{
				Namespace:  namespace,
				StripeCode: stripeCode,
			})
			switch {
			case taxcode.IsTaxCodeNotFoundError(err):
				continue
			case err != nil:
				return &productcataloghttp.TaxCodeResolutionError{Err: err}
			}

			cfg.TaxCodeID = tc.ID
		}

		if cfg.TaxCodeID == "" && cfg.Behavior == nil {
			continue
		}

		// The mapper rejects a line whose rate card config differs from its own, so both get it.
		lines[i].TaxConfig = lo.ToPtr(productcataloghttp.FromTaxConfig(cfg.ToTaxConfig()))
		if lines[i].RateCard != nil {
			lines[i].RateCard.TaxConfig = lines[i].TaxConfig
		}
	}

	return billing.ValidateRequestTaxCodeIDs(ctx, billing.ValidateRequestTaxCodeIDsInput{
		Namespace:  namespace,
		Resolver:   resolver,
		References: requestRefs,
	})
}
