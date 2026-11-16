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

// resolveTaxCodes prepares the tax inputs of invoice calculation: the invoice's default tax config
// in the legacy line shape, the default's Tax Code by ID, and the Tax Codes of the other Stripe codes
// the lines reference, keyed by Stripe code.
//
// The default is loaded by ID only, deleted included: the invoice inherited it from the profile.
// Resolving it by Stripe mapping would skip the deleted code and create or pick another one.
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

			deps.DefaultTaxCode = defaultTaxCodes[cfg.TaxCodeID]
		}

		legacyDefault := cfg.ToLegacyTaxConfig(defaultTaxCodes)
		deps.DefaultTaxConfig = billing.FromProductCatalog(&legacyDefault)
	}

	for _, line := range invoice.Lines.OrEmpty() {
		cfg := line.TaxConfig
		if cfg == nil || cfg.Stripe == nil || cfg.Stripe.Code == "" {
			continue
		}

		// Lines that already inherited the default get its Tax Code by ID.
		if deps.DefaultTaxCode != nil && cfg.TaxCodeID != nil && *cfg.TaxCodeID == deps.DefaultTaxCode.ID {
			continue
		}

		seen[cfg.Stripe.Code] = struct{}{}
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
