package productcatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type NamespacedFeatureResolver interface {
	Resolve(ctx context.Context, id, key *string) (*feature.Feature, error)
	WithNamespace(namespace string) NamespacedFeatureResolver
}

func NewNamespacedFeatureResolver(service feature.FeatureConnector, namespace string) NamespacedFeatureResolver {
	r := &namespacedFeatureResolver{
		service:   service,
		namespace: namespace,
	}

	return r
}

var _ NamespacedFeatureResolver = (*namespacedFeatureResolver)(nil)

type namespacedFeatureResolver struct {
	service   feature.FeatureConnector
	namespace string
}

func (r *namespacedFeatureResolver) WithNamespace(namespace string) NamespacedFeatureResolver {
	return &namespacedFeatureResolver{
		service:   r.service,
		namespace: namespace,
	}
}

func (r *namespacedFeatureResolver) Resolve(ctx context.Context, id, key *string) (*feature.Feature, error) {
	if r.service == nil {
		return nil, errors.New("feature connector is not set")
	}

	var featureIDOrKey []string

	if id != nil && *id != "" {
		featureIDOrKey = append(featureIDOrKey, *id)
	}

	if key != nil && *key != "" {
		featureIDOrKey = append(featureIDOrKey, *key)
	}

	if len(featureIDOrKey) == 0 {
		return nil, errors.New("either feature id or key must be provided")
	}

	features, err := r.service.ListFeatures(ctx, feature.ListFeaturesParams{
		IDsOrKeys:       featureIDOrKey,
		Namespace:       r.namespace,
		IncludeArchived: false,
		Page: pagination.Page{
			PageSize:   100,
			PageNumber: 1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feature: %w", err)
	}

	if len(features.Items) == 0 {
		return nil, models.NewGenericNotFoundError(errors.New("feature"))
	}

	var f feature.Feature

	m := make(map[string]struct{})

	for _, f = range features.Items {
		m[f.ID] = struct{}{}
	}

	if len(m) != 1 {
		return nil, models.NewGenericConflictError(fmt.Errorf("id and key reference %d different features", len(m)))
	}

	return &f, nil
}
