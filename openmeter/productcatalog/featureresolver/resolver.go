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

func (n *namespacedResolver) Resolve(ctx context.Context, reference feature.FeatureReference) (*feature.Feature, error) {
	return n.resolver.Resolve(ctx, n.namespace, reference)
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

func (r *resolver) Resolve(ctx context.Context, namespace string, reference feature.FeatureReference) (*feature.Feature, error) {
	if err := reference.Validate(); err != nil {
		return nil, fmt.Errorf("invalid feature reference: %w", err)
	}

	batch := make([]string, 0, 2)

	if reference.ID != nil {
		batch = append(batch, *reference.ID)
	}

	if reference.Key != nil {
		batch = append(batch, *reference.Key)
	}

	features, err := r.BatchResolve(ctx, namespace, batch...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feature: %w", err)
	}

	var f *feature.Feature

	if reference.ID != nil {
		f = features[*reference.ID]

		if f == nil {
			return nil, models.NewGenericNotFoundError(fmt.Errorf("feature [feature.id=%s]", lo.FromPtr(reference.ID)))
		}
	}

	if reference.Key != nil {
		if f == nil {
			f = features[*reference.Key]
		}

		if f == nil {
			return nil, models.NewGenericNotFoundError(fmt.Errorf("feature [feature.key=%s]", lo.FromPtr(reference.Key)))
		}

		if features[*reference.Key] == nil {
			return nil, models.NewGenericNotFoundError(fmt.Errorf("feature [feature.key=%s]", lo.FromPtr(reference.Key)))
		}
	}

	if _, err := reference.WithFeature(f); err != nil {
		return nil, models.NewGenericConflictError(fmt.Errorf("feature [feature.id=%s feature.key=%s]: %w", lo.FromPtr(reference.ID), lo.FromPtr(reference.Key), err))
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
