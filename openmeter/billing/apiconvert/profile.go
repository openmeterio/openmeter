package apiconvert

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
)

// LegacyProfileDefaultTaxConfig renders a stored default tax config the way the API shows it,
// so the deprecation gate can compare the request against what the client last read.
func LegacyProfileDefaultTaxConfig(ctx context.Context, resolver productcatalog.TaxCodeResolver, namespace string, workflow billing.WorkflowConfig) (*productcatalog.TaxConfig, error) {
	cfg := workflow.Invoicing.DefaultTaxConfig
	if cfg == nil {
		return nil, nil
	}

	taxCodes, err := ResolveWorkflowTaxCodes(ctx, resolver, namespace, workflow)
	if err != nil {
		return nil, err
	}

	return lo.ToPtr(cfg.ToLegacyTaxConfig(taxCodes)), nil
}

// ResolveProfileDefaultTaxConfig turns a billing profile's raw default tax config from a v1 or v3
// request into a Tax Code reference. raw and stored are in the legacy shape; stored is the profile's
// current default as responses show it (nil on create).
//
// It runs the deprecation gate first, so only removals, behavior changes and echoes of the stored
// tax code get through. taxCodeId wins over stripe.code. A stripe.code echoed without taxCodeId
// can only be the stored one, so it maps to the stored Tax Code ID: looking it up by Stripe mapping
// could return another Tax Code with the same mapping, or a new one when the stored code was
// deleted, and the service would then reject the no-op update.
//
// Errors are a billing.ValidationError for bad input, and a TaxCodeResolutionError (passed
// through, so it's a 500) when resolving a Stripe code fails. Call it inside
// TransactionalTaxCodeResolver.WithResolver, together with the service call.
func ResolveProfileDefaultTaxConfig(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver, namespace string, raw, stored *productcatalog.TaxConfig) (*productcatalog.TaxCodeConfig, error) {
	if err := raw.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("invalid default tax config: %w", err),
		}
	}

	normalized, err := billing.WithDeprecatedDefaultTaxCodeEnforced(raw, stored)
	if err != nil {
		return nil, err
	}

	if normalized == nil {
		return nil, nil
	}

	storedTaxCodeID := lo.FromPtr(lo.FromPtr(stored).TaxCodeID)
	if normalized.TaxCodeID == nil && lo.FromPtr(normalized.Stripe).Code != "" && storedTaxCodeID != "" {
		normalized.TaxCodeID = lo.ToPtr(storedTaxCodeID)
	}

	cfg, err := productcataloghttp.FromAPITaxConfig(ctx, resolver, namespace, productcataloghttp.FromTaxConfig(*normalized))
	if err != nil {
		var resolveErr *productcataloghttp.TaxCodeResolutionError
		if errors.As(err, &resolveErr) {
			return nil, err
		}

		return nil, billing.ValidationError{Err: err}
	}

	return cfg, nil
}
