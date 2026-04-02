package customerbalance

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type CurrencyFilter struct {
	Codes []currencyx.Code
}

func (f CurrencyFilter) Validate() error {
	for _, code := range f.Codes {
		if code == "" {
			return errors.New("currency code is required")
		}
	}

	return nil
}

type GetBalancesInput struct {
	CustomerID customer.CustomerID
	Currencies CurrencyFilter
}

func (i GetBalancesInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if err := i.Currencies.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currencies: %w", err))
	}

	return errors.Join(errs...)
}

type BalanceByCurrency struct {
	Currency currencyx.Code
	Balance  ledger.Balance
}

type Facade struct {
	service *Service
}

func NewFacade(service *Service) (*Facade, error) {
	if service == nil {
		return nil, errors.New("service is required")
	}

	return &Facade{
		service: service,
	}, nil
}

func (f *Facade) GetBalances(ctx context.Context, input GetBalancesInput) ([]BalanceByCurrency, error) {
	if f == nil {
		return nil, errors.New("facade is required")
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	var codes []currencyx.Code
	if len(input.Currencies.Codes) > 0 {
		codes = dedupeCurrencies(input.Currencies.Codes)

		for _, code := range codes {
			if err := code.Validate(); err != nil {
				return nil, fmt.Errorf("currency %q is not supported by ledger: %w", code, err)
			}
		}
	} else {
		var err error

		codes, err = f.service.getFBOCurrencies(ctx, input.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("get FBO currencies: %w", err)
		}
	}

	balances := make([]BalanceByCurrency, 0, len(codes))
	for _, code := range codes {
		balance, err := f.service.GetBalance(ctx, input.CustomerID, routeFilter(code))
		if err != nil {
			return nil, err
		}

		balances = append(balances, BalanceByCurrency{
			Currency: code,
			Balance:  balance,
		})
	}

	return balances, nil
}

func routeFilter(currency currencyx.Code) ledger.RouteFilter {
	return ledger.RouteFilter{
		Currency: currency,
	}
}

func dedupeCurrencies(codes []currencyx.Code) []currencyx.Code {
	seen := make(map[currencyx.Code]struct{}, len(codes))
	out := make([]currencyx.Code, 0, len(codes))

	for _, code := range codes {
		if _, ok := seen[code]; ok {
			continue
		}

		seen[code] = struct{}{}
		out = append(out, code)
	}

	return out
}
