package productcatalog

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

type FeatureResolver interface {
	Resolve(ctx context.Context, namespace string, id, key *string) (*feature.Feature, error)
	BatchResolve(ctx context.Context, namespace string, idOrKeys ...string) (map[string]*feature.Feature, error)
	WithNamespace(namespace string) NamespacedFeatureResolver
}

type NamespacedFeatureResolver interface {
	Resolve(ctx context.Context, id, key *string) (*feature.Feature, error)
	BatchResolve(ctx context.Context, idOrKeys ...string) (map[string]*feature.Feature, error)
	Namespace() string
}
