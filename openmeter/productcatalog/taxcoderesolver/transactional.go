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

// NewTransactional opens WithResolver's transaction on txCreator; the domain service called
// inside fn joins it through ctx. Pass the tax code repository.
func NewTransactional(service taxcode.Service, txCreator transaction.Creator) (productcatalog.TransactionalTaxCodeResolver, error) {
	if txCreator == nil {
		return nil, errors.New("transaction creator is not set")
	}

	readResolver, err := New(service)
	if err != nil {
		return nil, err
	}

	return &transactionalResolver{
		TaxCodeResolver: readResolver,
		txCreator:       txCreator,
		backfillResolver: &backfillResolver{
			service: service,
		},
	}, nil
}

var _ productcatalog.TransactionalTaxCodeResolver = (*transactionalResolver)(nil)

type transactionalResolver struct {
	productcatalog.TaxCodeResolver

	txCreator        transaction.Creator
	backfillResolver *backfillResolver
}

func (r *transactionalResolver) WithResolver(ctx context.Context, fn func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error) error {
	return transaction.RunWithNoValue(ctx, r.txCreator, func(ctx context.Context) error {
		return fn(ctx, r.backfillResolver)
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
