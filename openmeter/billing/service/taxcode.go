package billingservice

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

type resolveTaxCodesInput struct {
	Namespace string
	Invoice   *billing.StandardInvoice
}

func (i resolveTaxCodesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.Invoice == nil {
		errs = append(errs, errors.New("invoice is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// resolveTaxCodes prepares the tax inputs of invoice calculation: the invoice's default tax
// config (which the calculator merges into every line), the Tax Codes of the default and of
// every line (soft-deleted included) and the type of the invoicing app, whose provider code the
// calculator stamps on the lines.
func (s *Service) resolveTaxCodes(ctx context.Context, in resolveTaxCodesInput) (invoicecalc.StandardInvoiceCalculatorDependencies, error) {
	var deps invoicecalc.StandardInvoiceCalculatorDependencies

	if err := in.Validate(); err != nil {
		return deps, err
	}

	apps := in.Invoice.Workflow.Apps
	if apps == nil || apps.Invoicing == nil {
		return deps, fmt.Errorf("resolving the invoicing app type of invoice [%s]: apps are not expanded", in.Invoice.ID)
	}

	deps.InvoicingAppType = apps.Invoicing.GetType()

	cfg := in.Invoice.Workflow.Config.Invoicing.DefaultTaxConfig

	var ids []string

	if cfg != nil && cfg.TaxCodeID != "" {
		ids = append(ids, cfg.TaxCodeID)
	}

	for _, line := range in.Invoice.Lines.OrEmpty() {
		if line.TaxConfig != nil && line.TaxConfig.TaxCodeID != "" {
			ids = append(ids, line.TaxConfig.TaxCodeID)
		}
	}

	ids = lo.Uniq(ids)

	if len(ids) > 0 {
		taxCodes, err := s.taxCodeResolver.BatchResolve(ctx, in.Namespace, ids...)
		if err != nil {
			return deps, fmt.Errorf("resolving invoice tax codes: %w", err)
		}

		// A default pointing at a Tax Code outside the namespace (or a missing one) is corrupt.
		if cfg != nil && cfg.TaxCodeID != "" {
			if _, ok := taxCodes.Get(cfg.TaxCodeID); !ok {
				return deps, fmt.Errorf("resolving the invoice default tax code: %w", taxcode.NewTaxCodeNotFoundError(cfg.TaxCodeID))
			}
		}

		deps.TaxCodes = taxCodes
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
