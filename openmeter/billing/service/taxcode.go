package billingservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

type resolveTaxCodesInput struct {
	Namespace string
	Invoice   *billing.StandardInvoice
	// ReadOnly switches between a read-only lookup (GetTaxCodeByAppMapping, for
	// preview/simulate flows that must not write) and GetOrCreateByAppMapping
	// (for real write flows in the invoice state machine).
	ReadOnly bool
}

// resolveTaxCodes prepares the tax inputs of invoice calculation: it batch-resolves all unique
// Stripe tax codes referenced in the invoice's lines into a TaxCodes map keyed by Stripe code, and
// renders the invoice's default tax config in the legacy line shape.
//
// The default only stores a Tax Code ID, so its Tax Code is looked up by ID (deleted ones
// included: the invoice may still reference one) and its Stripe code comes from that Tax Code's
// mapping. That way lines inheriting the default still end up with the Stripe code, Tax Code and
// Tax Code ID they always got. The default never creates a Tax Code.
func (s *Service) resolveTaxCodes(ctx context.Context, in resolveTaxCodesInput) (invoicecalc.StandardInvoiceCalculatorDependencies, error) {
	namespace := in.Namespace
	invoice := in.Invoice
	readOnly := in.ReadOnly

	var deps invoicecalc.StandardInvoiceCalculatorDependencies

	result := make(invoicecalc.TaxCodes)
	// Unique Stripe codes to resolve by mapping.
	seen := make(map[string]struct{})

	if cfg := invoice.Workflow.Config.Invoicing.DefaultTaxConfig; cfg != nil {
		var defaultTaxCodes map[string]*taxcode.TaxCode

		if cfg.TaxCodeID != "" {
			var err error

			defaultTaxCodes, err = s.taxCodeResolver.BatchResolve(ctx, namespace, cfg.TaxCodeID)
			if err != nil {
				return deps, fmt.Errorf("resolving default tax code %q: %w", cfg.TaxCodeID, err)
			}

			// A row pointing at a Tax Code outside the namespace (or a missing one) is corrupt.
			if defaultTaxCodes[cfg.TaxCodeID] == nil {
				return deps, taxcode.NewTaxCodeNotFoundError(cfg.TaxCodeID)
			}
		}

		legacyDefault := cfg.ToLegacyTaxConfig(defaultTaxCodes)
		deps.DefaultTaxConfig = billing.FromProductCatalog(&legacyDefault)

		// The default's Tax Code is stamped onto the lines that use its Stripe code. A deleted one
		// isn't: those Stripe codes resolve by mapping to a live Tax Code, as lines carrying the
		// code themselves always did.
		if legacyDefault.Stripe != nil {
			if tc := defaultTaxCodes[cfg.TaxCodeID]; !tc.IsDeleted() {
				result[legacyDefault.Stripe.Code] = *tc
			} else {
				seen[legacyDefault.Stripe.Code] = struct{}{}
			}
		}
	}

	for _, line := range invoice.Lines.OrEmpty() {
		if line.TaxConfig != nil && line.TaxConfig.Stripe != nil && line.TaxConfig.Stripe.Code != "" {
			if _, ok := result[line.TaxConfig.Stripe.Code]; ok {
				continue
			}

			seen[line.TaxConfig.Stripe.Code] = struct{}{}
		}
	}

	for stripeCode := range seen {
		if readOnly {
			tc, err := s.taxCodeService.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput{
				Namespace: namespace,
				AppType:   app.AppTypeStripe,
				TaxCode:   stripeCode,
			})
			if err != nil {
				if taxcode.IsTaxCodeNotFoundError(err) {
					continue
				}

				return deps, fmt.Errorf("resolving tax code %q: %w", stripeCode, err)
			}

			result[stripeCode] = tc
		} else {
			tc, err := s.taxCodeService.GetOrCreateByAppMapping(ctx, taxcode.GetOrCreateByAppMappingInput{
				Namespace: namespace,
				AppType:   app.AppTypeStripe,
				TaxCode:   stripeCode,
			})
			if err != nil {
				return deps, fmt.Errorf("resolving tax code %q: %w", stripeCode, err)
			}

			result[stripeCode] = tc
		}
	}

	if len(result) > 0 {
		deps.TaxCodes = result
	}

	return deps, nil
}
