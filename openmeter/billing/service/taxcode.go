package billingservice

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

type resolveTaxCodesInput struct {
	Namespace string
	Invoice   *billing.StandardInvoice
}

// resolveTaxCodes prepares the tax inputs of invoice calculation: the invoice's default tax
// config, which the calculator merges into every line. Lines already carry their Tax Code ID:
// authored-reference validation and the FK keep those IDs valid, and the migrations gave every
// line with a Stripe code an ID, so there is nothing else to look up here.
func (s *Service) resolveTaxCodes(ctx context.Context, in resolveTaxCodesInput) (invoicecalc.StandardInvoiceCalculatorDependencies, error) {
	var deps invoicecalc.StandardInvoiceCalculatorDependencies

	cfg := in.Invoice.Workflow.Config.Invoicing.DefaultTaxConfig
	if cfg == nil {
		return deps, nil
	}

	if cfg.TaxCodeID != "" {
		// A default pointing at a Tax Code outside the namespace (or a missing one) is corrupt.
		taxCodes, err := s.taxCodeResolver.BatchResolve(ctx, in.Namespace, cfg.TaxCodeID)
		if err != nil {
			return deps, fmt.Errorf("resolving the invoice default tax code: %w", err)
		}

		if _, ok := taxCodes.Get(cfg.TaxCodeID); !ok {
			return deps, fmt.Errorf("resolving the invoice default tax code: %w", taxcode.NewTaxCodeNotFoundError(cfg.TaxCodeID))
		}
	}

	deps.DefaultTaxConfig = cfg

	return deps, nil
}

// lineTaxCodes loads the Tax Codes (soft-deleted included) referenced by the given line configs,
// so system events can render the legacy taxOverrides payload.
func (s *Service) lineTaxCodes(ctx context.Context, namespace string, configs ...*productcatalog.TaxCodeConfig) (productcatalog.TaxCodesByID, error) {
	seen := make(map[string]struct{}, len(configs))

	for _, cfg := range configs {
		if cfg != nil && cfg.TaxCodeID != "" {
			seen[cfg.TaxCodeID] = struct{}{}
		}
	}

	if len(seen) == 0 {
		return nil, nil
	}

	ids := slices.Collect(maps.Keys(seen))

	taxCodes, err := s.taxCodeResolver.BatchResolve(ctx, namespace, ids...)
	if err != nil {
		return nil, fmt.Errorf("resolving line tax codes: %w", err)
	}

	return taxCodes, nil
}
