package transactions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ConvertCurrencyTemplate materializes a fiat-funded custom-currency
// receivable exchange. The source is fiat and the target is custom currency.
type ConvertCurrencyTemplate struct {
	At           time.Time
	SourceAmount alpacadecimal.Decimal
	TargetAmount alpacadecimal.Decimal
	CostBasis    alpacadecimal.Decimal

	SourceCurrency currencies.CurrencyReference
	TargetCurrency currencies.CurrencyReference
	Features       []string
	SourceChargeID *string
	SpendChargeID  *string
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
	} else if !t.TargetCurrency.IsCustom() {
		errs = append(errs, errors.New("target currency must be custom"))
	} else if !t.TargetCurrency.IsResolved() {
		errs = append(errs, errors.New("target custom currency must be resolved"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ CustomerTransactionTemplate = (ConvertCurrencyTemplate{})

// correct reverses the complete original conversion, reusing its exact rounded
// source amount. Partial correction requires persisted cross-realization FX
// remainder state and is therefore intentionally unsupported here.
func (t ConvertCurrencyTemplate) correct(scope CorrectionInput) ([]ledger.TransactionInput, error) {
	if scope.CostBasis == nil {
		return nil, errors.New("cost basis is required to correct a currency conversion")
	}

	var sourceReceivable, brokerageSource, targetReceivable, brokerageTarget ledger.Entry
	var originalSourceAmount, originalTargetAmount alpacadecimal.Decimal
	var originalCostBasis *alpacadecimal.Decimal

	for _, entry := range scope.OriginalTransaction.Entries() {
		route := entry.PostingAddress().Route().Route()

		switch {
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerReceivable && entry.Amount().IsNegative():
			sourceReceivable = entry
			originalSourceAmount = originalSourceAmount.Add(entry.Amount().Abs())
			originalCostBasis = route.CostBasis
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerReceivable && entry.Amount().IsPositive():
			targetReceivable = entry
			originalTargetAmount = originalTargetAmount.Add(entry.Amount())
		case entry.PostingAddress().AccountType() == ledger.AccountTypeBrokerage && entry.Amount().IsPositive() && route.CostBasisCurrency == nil:
			brokerageSource = entry
		case entry.PostingAddress().AccountType() == ledger.AccountTypeBrokerage && entry.Amount().IsNegative() && route.CostBasisCurrency != nil:
			brokerageTarget = entry
		}
	}

	if sourceReceivable == nil || brokerageSource == nil || targetReceivable == nil || brokerageTarget == nil {
		return nil, errors.New("currency conversion correction requires the original source and target receivable and brokerage entries")
	}

	if originalCostBasis == nil || !originalCostBasis.Equal(*scope.CostBasis) {
		return nil, fmt.Errorf("correction cost basis %s does not match the original booking", scope.CostBasis.String())
	}

	if !scope.Amount.Equal(originalTargetAmount) {
		return nil, fmt.Errorf("currency conversion correction requires full original target amount %s, got %s", originalTargetAmount.String(), scope.Amount.String())
	}

	entryInputs := []*EntryInput{
		{
			address: sourceReceivable.PostingAddress(),
			amount:  originalSourceAmount,
			identity: ledger.EntryIdentityParts{
				SourceChargeID: sourceReceivable.SourceChargeID(),
				SpendChargeID:  sourceReceivable.SpendChargeID(),
			},
		},
		{
			address: brokerageSource.PostingAddress(),
			amount:  originalSourceAmount.Neg(),
			identity: ledger.EntryIdentityParts{
				SourceChargeID: brokerageSource.SourceChargeID(),
				SpendChargeID:  brokerageSource.SpendChargeID(),
			},
		},
		{
			address: targetReceivable.PostingAddress(),
			amount:  scope.Amount.Neg(),
			identity: ledger.EntryIdentityParts{
				SourceChargeID: targetReceivable.SourceChargeID(),
				SpendChargeID:  targetReceivable.SpendChargeID(),
			},
		},
		{
			address: brokerageTarget.PostingAddress(),
			amount:  scope.Amount,
			identity: ledger.EntryIdentityParts{
				SourceChargeID: brokerageTarget.SourceChargeID(),
				SpendChargeID:  brokerageTarget.SpendChargeID(),
			},
		},
	}

	return []ledger.TransactionInput{
		&TransactionInput{
			bookedAt:    scope.At,
			entryInputs: entryInputs,
		},
	}, nil
}

func (t ConvertCurrencyTemplate) typeGuard() guard {
	return true
}

func (t ConvertCurrencyTemplate) code() TransactionTemplateCode {
	return TemplateCodeConvertCurrency
}

func (t ConvertCurrencyTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	identity := ledger.EntryIdentityParts{
		SourceChargeID: t.SourceChargeID,
		SpendChargeID:  t.SpendChargeID,
	}
	costBasis := t.CostBasis
	targetCostBasisCurrency := t.SourceCurrency.Code
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
		CostBasisCurrency:              &targetCostBasisCurrency,
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
		CostBasisCurrency: &targetCostBasisCurrency,
		CostBasis:         &costBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brokerage target sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			// Source currency
			{
				address:  sourceAccount.Address(),
				amount:   t.SourceAmount.Neg(),
				identity: identity,
			},
			{
				address:  brokerageSource.Address(),
				amount:   t.SourceAmount,
				identity: identity,
			},
			// Target currency
			{
				address:  targetAccount.Address(),
				amount:   t.TargetAmount,
				identity: identity,
			},
			{
				address:  brokerageTarget.Address(),
				amount:   t.TargetAmount.Neg(),
				identity: identity,
			},
		},
	}, nil
}
