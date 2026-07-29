package transactions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	goblcurrency "github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ConvertCurrencyTemplate struct {
	At           time.Time
	SourceAmount alpacadecimal.Decimal
	TargetAmount alpacadecimal.Decimal
	CostBasis    alpacadecimal.Decimal

	SourceCurrency currencyx.Code
	TargetCurrency currencyx.Code
	// TargetCustomCurrency is the managed identity of TargetCurrency, which is
	// always a custom currency (see Validate).
	TargetCustomCurrency *ledger.CustomCurrencyIdentity
	Features             []string
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

	var sourceCurrencyDefinition *goblcurrency.Def
	if err := ledger.ValidateCurrency(t.SourceCurrency); err != nil {
		errs = append(errs, fmt.Errorf("source currency: %w", err))
	} else if sourceCurrencyDefinition = goblcurrency.Get(goblcurrency.Code(t.SourceCurrency)); sourceCurrencyDefinition == nil {
		errs = append(errs, errors.New("source currency must be a known fiat currency"))
	}

	if err := ledger.ValidateCurrency(t.TargetCurrency); err != nil {
		errs = append(errs, fmt.Errorf("target currency: %w", err))
	} else if goblcurrency.Get(goblcurrency.Code(t.TargetCurrency)) != nil {
		errs = append(errs, errors.New("target currency must be custom"))
	}

	if err := ledger.ValidateCustomCurrency(t.TargetCurrency, t.TargetCustomCurrency); err != nil {
		errs = append(errs, fmt.Errorf("target custom currency: %w", err))
	}

	if sourceAmountErr == nil && targetAmountErr == nil && costBasisErr == nil && sourceCurrencyDefinition != nil {
		expectedSourceAmount := t.TargetAmount.Mul(t.CostBasis).RoundBank(int32(sourceCurrencyDefinition.Subunits))
		if !t.SourceAmount.Equal(expectedSourceAmount) {
			errs = append(errs, fmt.Errorf(
				"source amount: expected %s from target amount multiplied by cost basis, got %s",
				expectedSourceAmount.String(),
				t.SourceAmount.String(),
			))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

var _ CustomerTransactionTemplate = (ConvertCurrencyTemplate{})

// correct reverses scope.Amount (denominated in the target/custom currency)
// of an original currency conversion. scope.CostBasis must match the cost
// basis the original conversion booked at; it is validated, not
// reverse-engineered, because the rate is immutable for a given purchase.
// A correction that reverses the whole original target amount reuses the
// exact original source amount instead of recomputing it, so it absorbs any
// rounding difference the original booking's precision left behind. Partial
// corrections recompute the proportional source amount from the supplied
// cost basis.
func (t ConvertCurrencyTemplate) correct(scope CorrectionInput) ([]ledger.TransactionInput, error) {
	if scope.CostBasis == nil {
		return nil, fmt.Errorf("cost basis is required to correct a currency conversion")
	}

	var sourceReceivable, brokerageSource, targetReceivable, brokerageTarget ledger.PostingAddress
	var originalSourceAmount, originalTargetAmount alpacadecimal.Decimal
	var originalCostBasis *alpacadecimal.Decimal

	for _, entry := range scope.OriginalTransaction.Entries() {
		route := entry.PostingAddress().Route().Route()
		switch {
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerReceivable && entry.Amount().IsNegative():
			sourceReceivable = entry.PostingAddress()
			originalSourceAmount = originalSourceAmount.Add(entry.Amount().Abs())
			originalCostBasis = route.CostBasis
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerReceivable && entry.Amount().IsPositive():
			targetReceivable = entry.PostingAddress()
			originalTargetAmount = originalTargetAmount.Add(entry.Amount())
		case entry.PostingAddress().AccountType() == ledger.AccountTypeBrokerage && entry.Amount().IsPositive() && route.CostBasisCurrency == nil:
			brokerageSource = entry.PostingAddress()
		case entry.PostingAddress().AccountType() == ledger.AccountTypeBrokerage && entry.Amount().IsNegative() && route.CostBasisCurrency != nil:
			brokerageTarget = entry.PostingAddress()
		}
	}

	if sourceReceivable == nil || brokerageSource == nil || targetReceivable == nil || brokerageTarget == nil {
		return nil, fmt.Errorf("currency conversion correction requires the original source/target receivable and brokerage entries")
	}

	if originalCostBasis == nil || !originalCostBasis.Equal(*scope.CostBasis) {
		return nil, fmt.Errorf("correction cost basis %s does not match the original booking", scope.CostBasis.String())
	}

	if scope.Amount.GreaterThan(originalTargetAmount) {
		return nil, fmt.Errorf("currency conversion correction amount %s exceeds original target amount %s", scope.Amount.String(), originalTargetAmount.String())
	}

	sourceCorrectionAmount := originalSourceAmount
	if !scope.Amount.Equal(originalTargetAmount) {
		sourceCurrencyDefinition := goblcurrency.Get(goblcurrency.Code(sourceReceivable.Route().Route().Currency))
		if sourceCurrencyDefinition == nil {
			return nil, fmt.Errorf("original source currency is not a known fiat currency")
		}

		sourceCorrectionAmount = scope.Amount.Mul(*scope.CostBasis).RoundBank(int32(sourceCurrencyDefinition.Subunits))
		if sourceCorrectionAmount.GreaterThan(originalSourceAmount) {
			return nil, fmt.Errorf("currency conversion correction amount %s exceeds original source amount %s", sourceCorrectionAmount.String(), originalSourceAmount.String())
		}
	}

	return []ledger.TransactionInput{
		&TransactionInput{
			bookedAt: scope.At,
			entryInputs: []*EntryInput{
				{
					address: sourceReceivable,
					amount:  sourceCorrectionAmount,
				},
				{
					address: brokerageSource,
					amount:  sourceCorrectionAmount.Neg(),
				},
				{
					address: targetReceivable,
					amount:  scope.Amount.Neg(),
				},
				{
					address: brokerageTarget,
					amount:  scope.Amount,
				},
			},
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
	costBasis := t.CostBasis
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
		CustomCurrency:                 t.TargetCustomCurrency,
		CostBasisCurrency:              lo.ToPtr(t.SourceCurrency),
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
		CustomCurrency:    t.TargetCustomCurrency,
		CostBasisCurrency: lo.ToPtr(t.SourceCurrency),
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
