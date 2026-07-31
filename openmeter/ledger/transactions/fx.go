package transactions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ConvertCurrencyTemplate struct {
	At           time.Time
	SourceAmount alpacadecimal.Decimal
	TargetAmount alpacadecimal.Decimal
	CostBasis    alpacadecimal.Decimal

	SourceCurrency currencies.CurrencyReference
	TargetCurrency currencies.CurrencyReference
	Features       []string
}

func (t ConvertCurrencyTemplate) Validate() error {
	var errs []error

	if t.At.IsZero() {
		errs = append(errs, errors.New("at is required"))
	}

	sourceAmountErr := ledger.ValidateTransactionAmount(t.SourceAmount)
	if sourceAmountErr != nil {
		errs = append(errs, fmt.Errorf("source amount: %w", sourceAmountErr))
	}

	targetAmountErr := ledger.ValidateTransactionAmount(t.TargetAmount)
	if targetAmountErr != nil {
		errs = append(errs, fmt.Errorf("target amount: %w", targetAmountErr))
	}

	costBasisErr := ledger.ValidateCostBasis(t.CostBasis)
	if costBasisErr != nil {
		errs = append(errs, fmt.Errorf("cost basis: %w", costBasisErr))
	} else if t.CostBasis.IsZero() {
		costBasisErr = errors.New("cost basis must be positive")
		errs = append(errs, costBasisErr)
	}

	if err := t.SourceCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("source currency: %w", err))
	} else if !t.SourceCurrency.IsFiat() {
		errs = append(errs, errors.New("source currency must be a known fiat currency"))
	}

	if err := t.TargetCurrency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("target currency: %w", err))
	} else if t.TargetCurrency.IsCustom() {
		if !t.TargetCurrency.IsResolved() {
			errs = append(errs, errors.New("target custom currency must be resolved"))
		}
	} else if !t.TargetCurrency.IsFiat() || t.TargetCurrency.Code != t.SourceCurrency.Code {
		errs = append(errs, errors.New("target currency must be custom or match the source fiat currency"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ CustomerTransactionTemplate = (ConvertCurrencyTemplate{})

func (t ConvertCurrencyTemplate) correct(CorrectionInput) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(TemplateCode(t))
}

func (t ConvertCurrencyTemplate) typeGuard() guard {
	return true
}

func (t ConvertCurrencyTemplate) code() TransactionTemplateCode {
	return TemplateCodeConvertCurrency
}

func (t ConvertCurrencyTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	costBasis := t.CostBasis
	targetCostBasisCurrency := lo.ToPtr(t.SourceCurrency.Code)
	if t.TargetCurrency.IsFiat() {
		targetCostBasisCurrency = nil
	}
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	sourceAccount, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.SourceCurrency,
		CostBasis:                      &costBasis,
		Features:                       t.Features,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get source sub-account: %w", err)
	}

	targetAccount, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.TargetCurrency,
		CostBasisCurrency:              targetCostBasisCurrency,
		CostBasis:                      &costBasis,
		Features:                       t.Features,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get target sub-account: %w", err)
	}

	businessAccounts, err := resolvers.AccountService.GetBusinessAccounts(ctx, customerID.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get business accounts: %w", err)
	}

	brokerageSource, err := businessAccounts.BrokerageAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
		Currency:  t.SourceCurrency,
		CostBasis: &costBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brokerage source sub-account: %w", err)
	}

	brokerageTarget, err := businessAccounts.BrokerageAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
		Currency:          t.TargetCurrency,
		CostBasisCurrency: targetCostBasisCurrency,
		CostBasis:         &costBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brokerage target sub-account: %w", err)
	}

	if t.TargetCurrency.IsFiat() {
		return &TransactionInput{
			bookedAt: t.At,
			entryInputs: []*EntryInput{
				{
					address: sourceAccount.Address(),
					amount:  t.TargetAmount.Sub(t.SourceAmount),
				},
				{
					address: brokerageSource.Address(),
					amount:  t.SourceAmount.Sub(t.TargetAmount),
				},
			},
		}, nil
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			// Source currency
			{
				address: sourceAccount.Address(),
				amount:  t.SourceAmount.Neg(),
			},
			{
				address: brokerageSource.Address(),
				amount:  t.SourceAmount,
			},
			// Target currency
			{
				address: targetAccount.Address(),
				amount:  t.TargetAmount,
			},
			{
				address: brokerageTarget.Address(),
				amount:  t.TargetAmount.Neg(),
			},
		},
	}, nil
}
