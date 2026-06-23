package taxcoderesolver

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

func New(service taxcode.Service) (productcatalog.TaxCodeResolver, error) {
	if service == nil {
		return nil, errors.New("tax code service is not set")
	}

	return &resolver{service: service}, nil
}

var _ productcatalog.NamespacedTaxCodeResolver = (*namespacedResolver)(nil)

type namespacedResolver struct {
	resolver  *resolver
	namespace string
}

func (n *namespacedResolver) Namespace() string { return n.namespace }

func (n *namespacedResolver) ResolveTaxCode(ctx context.Context, id string) (*taxcode.TaxCode, error) {
	return n.resolver.ResolveTaxCode(ctx, n.namespace, id)
}

var _ productcatalog.TaxCodeResolver = (*resolver)(nil)

type resolver struct {
	service taxcode.Service
}

func (r *resolver) WithNamespace(namespace string) productcatalog.NamespacedTaxCodeResolver {
	return &namespacedResolver{resolver: r, namespace: namespace}
}

func (r *resolver) ResolveTaxCode(ctx context.Context, namespace string, id string) (*taxcode.TaxCode, error) {
	if namespace == "" {
		return nil, errors.New("namespace is not set")
	}

	tc, err := r.service.GetTaxCode(ctx, taxcode.GetTaxCodeInput{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: id},
	})
	if err != nil {
		return nil, err
	}

	return &tc, nil
}
