package featureresolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// NOTE: this should live under the feature package after it gets refactored

func New(service feature.FeatureConnector) (productcatalog.FeatureResolver, error) {
	if service == nil {
		return nil, errors.New("feature connector is not set")
	}

	return &resolver{
		service: service,
	}, nil
}

var _ productcatalog.NamespacedFeatureResolver = (*namespacedResolver)(nil)

type namespacedResolver struct {
	resolver  *resolver
	namespace string
}

func (n *namespacedResolver) Namespace() string {
	return n.namespace
}

func (n *namespacedResolver) Resolve(ctx context.Context, id, key *string) (*feature.Feature, error) {
	return n.resolver.Resolve(ctx, n.namespace, id, key)
}

func (n *namespacedResolver) BatchResolve(ctx context.Context, idOrKeys ...string) (map[string]*feature.Feature, error) {
	return n.resolver.BatchResolve(ctx, n.namespace, idOrKeys...)
}

var _ productcatalog.FeatureResolver = (*resolver)(nil)

type resolver struct {
	service feature.FeatureConnector
}

func (r *resolver) WithNamespace(namespace string) productcatalog.NamespacedFeatureResolver {
	return &namespacedResolver{
		resolver:  r,
		namespace: namespace,
	}
}

func (r *resolver) Resolve(ctx context.Context, namespace string, id, key *string) (*feature.Feature, error) {
	hasID := id != nil && *id != ""
	hasKey := key != nil && *key != ""

	if !hasID && !hasKey {
		return nil, errors.New("feature id or key is required")
	}

	batch := make([]string, 0, 2)

	if hasID {
		batch = append(batch, *id)
	}

	if hasKey {
		batch = append(batch, *key)
	}

	features, err := r.BatchResolve(ctx, namespace, batch...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feature: %w", err)
	}

	var f *feature.Feature

	if hasID {
		f = features[*id]

		if f == nil {
			return nil, models.NewGenericNotFoundError(fmt.Errorf("feature [feature.id=%s]", lo.FromPtr(id)))
		}

		if f.ID != *id {
			return nil, models.NewGenericConflictError(fmt.Errorf("feature [feature.id=%s feature.key=%s]", lo.FromPtr(id), lo.FromPtr(key)))
		}
	}

	if hasKey {
		if f == nil {
			f = features[*key]
		}

		if f == nil {
			return nil, models.NewGenericNotFoundError(fmt.Errorf("feature [feature.key=%s]", lo.FromPtr(key)))
		}

		if features[*key] == nil {
			return nil, models.NewGenericNotFoundError(fmt.Errorf("feature [feature.key=%s]", lo.FromPtr(key)))
		}

		if f.Key != *key {
			return nil, models.NewGenericConflictError(fmt.Errorf("feature [feature.id=%s feature.key=%s]", lo.FromPtr(id), lo.FromPtr(key)))
		}
	}

	return f, nil
}

func (r *resolver) BatchResolve(ctx context.Context, namespace string, idsOrKeys ...string) (map[string]*feature.Feature, error) {
	if namespace == "" {
		return nil, errors.New("namespace is not set")
	}

	if len(idsOrKeys) == 0 {
		return nil, nil
	}

	features, err := pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[feature.Feature], error) {
		return r.service.ListFeatures(ctx, feature.ListFeaturesParams{
			IDsOrKeys:       idsOrKeys,
			Namespace:       namespace,
			IncludeArchived: false,
			Page:            page,
		})
	}), min(len(idsOrKeys), 100))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch features: %w", err)
	}

	result := lo.SliceToMap(idsOrKeys, func(item string) (string, *feature.Feature) {
		return item, nil
	})

	for idx := range features {
		f := features[idx]

		if _, ok := result[f.ID]; ok {
			result[f.ID] = &f
		}

		if _, ok := result[f.Key]; ok {
			result[f.Key] = &f
		}
	}

	return result, nil
}
