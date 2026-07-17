package currencies

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListCurrencies() ListCurrenciesHandler
	CreateCurrency() CreateCurrencyHandler
	CreateCostBasis() CreateCostBasisHandler
	ListCostBases() ListCostBasesHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	options          []httptransport.HandlerOption
	service          currencies.Service
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	currencyService currencies.Service,
	options ...httptransport.HandlerOption,
) (Handler, error) {
	var errs []error
	if resolveNamespace == nil {
		errs = append(errs, errors.New("namespace resolver is required"))
	}
	if currencyService == nil {
		errs = append(errs, errors.New("currency service is required"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid currency handler config: %w", err)
	}

	return &handler{
		resolveNamespace: resolveNamespace,
		options:          options,
		service:          currencyService,
	}, nil
}
