package billingservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// resolveTaxCodes collects all distinct TaxCodeId references from invoice lines and the default tax config,
// fetches the corresponding TaxCode entities, and returns them as a map keyed by ID.
func (s *Service) resolveTaxCodes(ctx context.Context, invoice billing.StandardInvoice) (map[string]taxcode.TaxCode, error) {
	if s.taxCodeService == nil {
		return nil, nil
	}

	taxCodeIDs := collectTaxCodeIDs(invoice)
	if len(taxCodeIDs) == 0 {
		return nil, nil
	}

	out := make(map[string]taxcode.TaxCode, len(taxCodeIDs))

	for _, id := range taxCodeIDs {
		tc, err := s.taxCodeService.GetTaxCode(ctx, taxcode.GetTaxCodeInput{
			NamespacedID: models.NamespacedID{
				Namespace: invoice.Namespace,
				ID:        id,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("resolving tax code %s: %w", id, err)
		}

		out[id] = tc
	}

	return out, nil
}

// collectTaxCodeIDs returns all distinct TaxCodeId values from the invoice's lines and default tax config.
func collectTaxCodeIDs(invoice billing.StandardInvoice) []string {
	seen := make(map[string]struct{})

	addIfSet := func(tc *productcatalog.TaxConfig) {
		if tc != nil && tc.TaxCodeId != nil {
			seen[*tc.TaxCodeId] = struct{}{}
		}
	}

	// Default tax config from workflow
	if invoice.Workflow.Config.Invoicing.DefaultTaxConfig != nil {
		addIfSet(invoice.Workflow.Config.Invoicing.DefaultTaxConfig)
	}

	// Line-level tax configs
	for _, line := range invoice.Lines.OrEmpty() {
		addIfSet(line.TaxConfig)
	}

	return lo.Keys(seen)
}
