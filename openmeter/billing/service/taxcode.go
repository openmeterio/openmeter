package billingservice

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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
// in the legacy line shape, the Tax Codes referenced by ID, and the Tax Codes of the Stripe codes
// lines reference without an ID, keyed by Stripe code.
//
// Everything referenced by ID — the inherited default and any line's own TaxCodeID — is loaded in
// one batch, deleted included: these references were validated when they were authored, and
// re-resolving them by Stripe mapping could create or pick another Tax Code. Only lines that
// identify their tax code purely by Stripe code keep the by-mapping lookup.
func (s *Service) resolveTaxCodes(ctx context.Context, in resolveTaxCodesInput) (invoicecalc.StandardInvoiceCalculatorDependencies, error) {
	namespace := in.Namespace
	invoice := in.Invoice
	readOnly := in.ReadOnly

	var deps invoicecalc.StandardInvoiceCalculatorDependencies

	result := make(invoicecalc.TaxCodes)
	// Unique Stripe codes to resolve by mapping.
	stripeCodes := make(map[string]struct{})

	// Unique Tax Code IDs to resolve by ID: the default's and the lines' own.
	taxCodeIDs := make(map[string]struct{})

	if cfg := invoice.Workflow.Config.Invoicing.DefaultTaxConfig; cfg != nil && cfg.TaxCodeID != "" {
		taxCodeIDs[cfg.TaxCodeID] = struct{}{}
	}

	for _, line := range invoice.Lines.OrEmpty() {
		cfg := line.TaxConfig
		if cfg == nil {
			continue
		}

		if cfg.TaxCodeID != nil && *cfg.TaxCodeID != "" {
			taxCodeIDs[*cfg.TaxCodeID] = struct{}{}

			continue
		}

		if cfg.Stripe == nil || cfg.Stripe.Code == "" {
			continue
		}

		stripeCodes[cfg.Stripe.Code] = struct{}{}
	}

	var taxCodesByID productcatalog.TaxCodesByID
	if len(taxCodeIDs) > 0 {
		var err error

		taxCodesByID, err = s.taxCodeResolver.BatchResolve(ctx, namespace, slices.Collect(maps.Keys(taxCodeIDs))...)
		if err != nil {
			return deps, fmt.Errorf("resolving tax codes by ID: %w", err)
		}
	}

	if cfg := invoice.Workflow.Config.Invoicing.DefaultTaxConfig; cfg != nil {
		var defaultTaxCodes map[string]*taxcode.TaxCode

		if cfg.TaxCodeID != "" {
			// A row pointing at a Tax Code outside the namespace (or a missing one) is corrupt.
			if taxCodesByID[cfg.TaxCodeID] == nil {
				return deps, taxcode.NewTaxCodeNotFoundError(cfg.TaxCodeID)
			}

			deps.DefaultTaxCode = taxCodesByID[cfg.TaxCodeID]

			defaultTaxCodes = map[string]*taxcode.TaxCode{cfg.TaxCodeID: deps.DefaultTaxCode}
		}

		legacyDefault := cfg.ToLegacyTaxConfig(defaultTaxCodes)
		deps.DefaultTaxConfig = billing.FromProductCatalog(&legacyDefault)
	}

	if len(taxCodeIDs) > 0 {
		deps.TaxCodesByID = make(productcatalog.TaxCodesByID, len(taxCodeIDs))
		for id := range taxCodeIDs {
			// A line ID missing from the result keeps the ID with no TaxCode attached: the
			// read side applies the same rule, and a missing row here means the reference
			// cannot be resolved (outside the namespace or gone), not that calculation
			// should fail for inherited history.
			deps.TaxCodesByID[id] = taxCodesByID[id]
		}
	}

	for stripeCode := range stripeCodes {
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
