package billingservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

// resolveTaxCodes batch-resolves all unique Stripe tax codes referenced in an invoice's lines
// and its DefaultTaxConfig, returning a TaxCodes map keyed by Stripe code.
// When readOnly is true it uses a read-only lookup (GetTaxCodeByAppMapping) and silently skips
// codes that have no stored entity yet — suitable for preview/simulate flows that must not write.
// When readOnly is false it uses GetOrCreateByAppMapping, which creates missing entities — suitable
// for real write flows (invoice state machine advancement).
func (s *Service) resolveTaxCodes(ctx context.Context, namespace string, invoice *billing.StandardInvoice, readOnly bool) (invoicecalc.TaxCodes, error) {
	// Collect unique Stripe codes from lines and the invoice-level default.
	seen := make(map[string]struct{})

	if invoice.Workflow.Config.Invoicing.DefaultTaxConfig != nil {
		if cfg := invoice.Workflow.Config.Invoicing.DefaultTaxConfig; cfg.Stripe != nil && cfg.Stripe.Code != "" {
			seen[cfg.Stripe.Code] = struct{}{}
		}
	}

	for _, line := range invoice.Lines.OrEmpty() {
		if line.TaxConfig != nil && line.TaxConfig.Stripe != nil && line.TaxConfig.Stripe.Code != "" {
			seen[line.TaxConfig.Stripe.Code] = struct{}{}
		}
	}

	if len(seen) == 0 {
		return nil, nil
	}

	result := make(invoicecalc.TaxCodes, len(seen))
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
				return nil, fmt.Errorf("resolving tax code %q: %w", stripeCode, err)
			}
			result[stripeCode] = tc
		} else {
			tc, err := s.taxCodeService.GetOrCreateByAppMapping(ctx, taxcode.GetOrCreateByAppMappingInput{
				Namespace: namespace,
				AppType:   app.AppTypeStripe,
				TaxCode:   stripeCode,
			})
			if err != nil {
				return nil, fmt.Errorf("resolving tax code %q: %w", stripeCode, err)
			}
			result[stripeCode] = tc
		}
	}

	return result, nil
}
