package productcatalog

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

// TaxCodeResolver resolves tax codes across namespaces and can create a namespace-scoped resolver.
type TaxCodeResolver interface {
	Resolve(ctx context.Context, namespace string, id string) (*taxcode.TaxCode, error)
	WithNamespace(namespace string) NamespacedTaxCodeResolver
}

// NamespacedTaxCodeResolver resolves tax codes within a fixed namespace.
type NamespacedTaxCodeResolver interface {
	Resolve(ctx context.Context, id string) (*taxcode.TaxCode, error)
	Namespace() string
}
