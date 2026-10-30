package productcatalog

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TaxCodeResolver looks up Tax Codes by ID within a namespace. It is read-only: it never
// creates Tax Codes and never resolves provider-specific codes.
type TaxCodeResolver interface {
	// Resolve returns a live Tax Code. Soft-deleted codes are reported as not found, so new
	// references can't point at them.
	Resolve(ctx context.Context, namespace string, id string) (*taxcode.TaxCode, error)
	// BatchResolve returns the Tax Codes found for ids, keyed by ID, including soft-deleted ones:
	// existing rows and issued invoices may still reference a deleted code. IDs that don't exist
	// in the namespace are missing from the map.
	BatchResolve(ctx context.Context, namespace string, ids ...string) (map[string]*taxcode.TaxCode, error)
	WithNamespace(namespace string) NamespacedTaxCodeResolver
}

// NamespacedTaxCodeResolver is a TaxCodeResolver bound to a single namespace.
type NamespacedTaxCodeResolver interface {
	Resolve(ctx context.Context, id string) (*taxcode.TaxCode, error)
	BatchResolve(ctx context.Context, ids ...string) (map[string]*taxcode.TaxCode, error)
	Namespace() string
}

// TaxCodeResolverFactory is what v1 handlers get: the read resolver for responses, plus
// WithResolver for turning legacy Stripe codes into Tax Codes. Services get only TaxCodeResolver,
// so they can't create Tax Codes.
type TaxCodeResolverFactory interface {
	TaxCodeResolver

	// WithResolver runs fn in one transaction. Tax Codes created by the resolver commit or roll
	// back together with whatever else fn writes, so a failed request doesn't leave behind
	// codes nothing references. Reads through the factory with fn's ctx see the codes created
	// in it.
	WithResolver(ctx context.Context, fn func(ctx context.Context, resolver TaxCodeBackfillResolver) error) error
}

// TaxCodeBackfillResolver resolves legacy Stripe codes to Tax Codes, creating missing ones. It's
// only handed out inside TaxCodeResolverFactory.WithResolver.
type TaxCodeBackfillResolver interface {
	// ResolveOrBackfillTaxCode returns the Tax Code mapped to the Stripe code, creating one if
	// the namespace doesn't have it yet.
	ResolveOrBackfillTaxCode(ctx context.Context, input ResolveOrBackfillTaxCodeInput) (*taxcode.TaxCode, error)
}

var _ models.Validator = (*ResolveOrBackfillTaxCodeInput)(nil)

type ResolveOrBackfillTaxCodeInput struct {
	Namespace  string
	StripeCode string
}

func (i ResolveOrBackfillTaxCodeInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.StripeCode == "" {
		errs = append(errs, errors.New("stripe code is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
