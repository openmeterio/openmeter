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

// resolveTaxCodes batch-resolves all unique Stripe tax codes referenced in an invoice's lines
// and its DefaultTaxConfig, returning a TaxCodes map keyed by Stripe code.
func (s *Service) resolveTaxCodes(ctx context.Context, in resolveTaxCodesInput) (invoicecalc.TaxCodes, error) {
	namespace := in.Namespace
	invoice := in.Invoice
	readOnly := in.ReadOnly
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
