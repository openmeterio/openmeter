package customerbalance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

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
	AsOf       *time.Time
}

type GetBalanceInput struct {
	CustomerID customer.CustomerID
	Currency   currencyx.Code
	After      *ledger.TransactionCursor
	AsOf       *time.Time
}

func (i GetBalancesInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if err := i.Currencies.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currencies: %w", err))
	}

	if i.AsOf != nil && i.AsOf.IsZero() {
		errs = append(errs, fmt.Errorf("asOf must not be zero"))
	}

	return errors.Join(errs...)
}

func (i GetBalanceInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if i.After != nil {
		if err := i.After.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("after: %w", err))
		}
	}

	if i.AsOf != nil && i.AsOf.IsZero() {
		errs = append(errs, fmt.Errorf("asOf must not be zero"))
	}

	if i.After != nil && i.AsOf != nil {
		errs = append(errs, fmt.Errorf("after and asOf cannot both be set"))
	}

	return errors.Join(errs...)
}

type BalanceByCurrency struct {
	Currency currencyx.Code
	Balance  ledger.Balance
}

type Facade struct {
	service Service
}

func NewFacade(service Service) (*Facade, error) {
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

		codes, err = f.service.GetFBOCurrencies(ctx, input.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("get FBO currencies: %w", err)
		}
	}

	balances := make([]BalanceByCurrency, 0, len(codes))
	for _, code := range codes {
		balance, err := f.service.GetBalance(ctx, input.CustomerID, code, ledger.BalanceQuery{
			AsOf: input.AsOf,
		})
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

func (f *Facade) GetBalance(ctx context.Context, input GetBalanceInput) (alpacadecimal.Decimal, error) {
	if f == nil {
		return alpacadecimal.Zero, errors.New("facade is required")
	}

	if err := input.Validate(); err != nil {
		return alpacadecimal.Zero, err
	}

	balance, err := f.service.GetBalance(ctx, input.CustomerID, input.Currency, ledger.BalanceQuery{
		After: input.After,
		AsOf:  input.AsOf,
	})
	if err != nil {
		return alpacadecimal.Zero, err
	}

	return balance.Settled(), nil
}

func (f *Facade) ListCreditTransactions(ctx context.Context, input ListCreditTransactionsInput) (ListCreditTransactionsResult, error) {
	if f == nil {
		return ListCreditTransactionsResult{}, errors.New("facade is required")
	}

	if err := input.Validate(); err != nil {
		return ListCreditTransactionsResult{}, err
	}

	return f.service.ListCreditTransactions(ctx, input)
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
