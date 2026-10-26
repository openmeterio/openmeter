package productcatalog

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
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
