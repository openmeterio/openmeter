package taxcoderesolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// NewFactory needs the tax code repository as the transaction creator: WithResolver opens the
// transaction on it, and the domain service called inside fn joins it through ctx.
func NewFactory(service taxcode.Service, txCreator taxcode.Repository) (productcatalog.TaxCodeResolverFactory, error) {
	if txCreator == nil {
		return nil, errors.New("tax code repository is not set")
	}

	readResolver, err := New(service)
	if err != nil {
		return nil, err
	}

	return &factory{
		TaxCodeResolver: readResolver,
		txCreator:       txCreator,
		backfillResolver: &backfillResolver{
			service: service,
		},
	}, nil
}

var _ productcatalog.TaxCodeResolverFactory = (*factory)(nil)

type factory struct {
	productcatalog.TaxCodeResolver

	txCreator        transaction.Creator
	backfillResolver *backfillResolver
}

func (f *factory) WithResolver(ctx context.Context, fn func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error) error {
	return transaction.RunWithNoValue(ctx, f.txCreator, func(ctx context.Context) error {
		return fn(ctx, f.backfillResolver)
	})
}

var _ productcatalog.TaxCodeBackfillResolver = (*backfillResolver)(nil)

type backfillResolver struct {
	service taxcode.Service
}

func (r *backfillResolver) ResolveOrBackfillTaxCode(ctx context.Context, input productcatalog.ResolveOrBackfillTaxCodeInput) (*taxcode.TaxCode, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	tc, err := r.service.GetOrCreateByAppMapping(ctx, taxcode.GetOrCreateByAppMappingInput{
		Namespace: input.Namespace,
		AppType:   app.AppTypeStripe,
		TaxCode:   input.StripeCode,
	})
	if err != nil {
		return nil, fmt.Errorf("resolving tax code for stripe code %s: %w", input.StripeCode, err)
	}

	return &tc, nil
}
