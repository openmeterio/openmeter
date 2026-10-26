package taxcoderesolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func New(service taxcode.Service) (productcatalog.TaxCodeResolver, error) {
	if service == nil {
		return nil, errors.New("tax code service is not set")
	}

	return &resolver{
		service: service,
	}, nil
}

var _ productcatalog.NamespacedTaxCodeResolver = (*namespacedResolver)(nil)

type namespacedResolver struct {
	resolver  *resolver
	namespace string
}

func (n *namespacedResolver) Namespace() string {
	return n.namespace
}

func (n *namespacedResolver) Resolve(ctx context.Context, id string) (*taxcode.TaxCode, error) {
	return n.resolver.Resolve(ctx, n.namespace, id)
}

func (n *namespacedResolver) BatchResolve(ctx context.Context, ids ...string) (map[string]*taxcode.TaxCode, error) {
	return n.resolver.BatchResolve(ctx, n.namespace, ids...)
}

var _ productcatalog.TaxCodeResolver = (*resolver)(nil)

type resolver struct {
	service taxcode.Service
}

func (r *resolver) WithNamespace(namespace string) productcatalog.NamespacedTaxCodeResolver {
	return &namespacedResolver{
		resolver:  r,
		namespace: namespace,
	}
}

func (r *resolver) Resolve(ctx context.Context, namespace string, id string) (*taxcode.TaxCode, error) {
	if namespace == "" {
		return nil, errors.New("namespace is not set")
	}

	tc, err := r.service.GetTaxCode(ctx, taxcode.GetTaxCodeInput{
		NamespacedID: models.NamespacedID{
			Namespace: namespace,
			ID:        id,
		},
	})
	if err != nil {
		return nil, err
	}

	return &tc, nil
}

func (r *resolver) BatchResolve(ctx context.Context, namespace string, ids ...string) (map[string]*taxcode.TaxCode, error) {
	if namespace == "" {
		return nil, errors.New("namespace is not set")
	}

	if len(ids) == 0 {
		return map[string]*taxcode.TaxCode{}, nil
	}

	taxCodes, err := pagination.CollectAll(ctx, pagination.NewPaginator(func(ctx context.Context, page pagination.Page) (pagination.Result[taxcode.TaxCode], error) {
		return r.service.ListTaxCodes(ctx, taxcode.ListTaxCodesInput{
			Namespace:      namespace,
			IDs:            ids,
			IncludeDeleted: true,
			Page:           page,
		})
	}), min(len(ids), 100))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tax codes: %w", err)
	}

	result := make(map[string]*taxcode.TaxCode, len(taxCodes))
	for idx := range taxCodes {
		result[taxCodes[idx].ID] = &taxCodes[idx]
	}

	return result, nil
}
