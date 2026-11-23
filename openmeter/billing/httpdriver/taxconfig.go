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

// resolvedLineTaxConfig pairs one request line with the tax config resolved for it.
type resolvedLineTaxConfig struct {
	// Line is the request line as the update received it.
	Line apiv1models.InvoiceLineReplaceUpdate
	// TaxConfig is the line's resolved tax config, nil when the line sent no tax config.
	TaxConfig *productcatalog.TaxConfig
}

// resolveUpdateLineTaxConfigs resolves every edited line's tax config up front, so the update's
// EditFn (which has no context) can apply the already-resolved IDs. Each result keeps the
// request line it belongs to, so the merge step fails loudly on an order mismatch instead of
// silently applying another line's config.
func resolveUpdateLineTaxConfigs(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver, namespace string, lines []apiv1models.InvoiceLineReplaceUpdate) ([]resolvedLineTaxConfig, error) {
	resolved := make([]resolvedLineTaxConfig, 0, len(lines))

	for _, line := range lines {
		raw := lineTaxConfig(line.RateCard, line.TaxConfig)
		if raw == nil {
			resolved = append(resolved, resolvedLineTaxConfig{Line: line})

			continue
		}

		rawConfig := productcataloghttp.AsTaxConfig(*raw)
		cfg, err := resolveLineTaxConfig(ctx, resolver, namespace, &rawConfig)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, resolvedLineTaxConfig{Line: line, TaxConfig: cfg})
	}

	return resolved, nil
}

// taxConfigsForRequestLines returns the resolved configs for the given request lines. The
// update's EditFn has no context, so the handler resolves up front and pairs each config with
// its line; the merge must see the same lines in the same order, and any mismatch is a caller
// bug that fails the request instead of silently applying another line's config.
func taxConfigsForRequestLines(resolved []resolvedLineTaxConfig, lines []apiv1models.InvoiceLineReplaceUpdate) ([]*productcatalog.TaxConfig, error) {
	if len(resolved) != len(lines) {
		return nil, fmt.Errorf("resolved tax configs (%d) do not match request lines (%d)", len(resolved), len(lines))
	}

	for i, entry := range resolved {
		if entry.Line.Id != nil && lines[i].Id != nil && *entry.Line.Id != *lines[i].Id {
			return nil, fmt.Errorf("resolved tax config at position %d belongs to line %q, not %q", i, lo.FromPtr(entry.Line.Id), lo.FromPtr(lines[i].Id))
		}
	}

	out := make([]*productcatalog.TaxConfig, len(resolved))
	for i, entry := range resolved {
		out[i] = entry.TaxConfig
	}

	return out, nil
}

// resolveSimulationLineTaxConfigs replaces each simulation line's stripe.code with the ID of the
// Tax Code it maps to. Simulation must not write, so the lookup is read-only: an unknown code
// leaves the line as sent, and the response shows it without stripe.code.
func resolveSimulationLineTaxConfigs(ctx context.Context, resolver productcatalog.TransactionalTaxCodeResolver, namespace string, lines []apiv1models.InvoiceSimulationLine) error {
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

	return nil
}
