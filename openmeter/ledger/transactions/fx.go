package transactions

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type ConvertCurrencyTemplate struct {
	At           time.Time
	TargetAmount alpacadecimal.Decimal
	CostBasis    alpacadecimal.Decimal

	SourceCurrency currencyx.Code
	TargetCurrency currencyx.Code
	// Optional, defaults to ledger.DefaultCustomerFBOPriority.
	CreditPriority *int
}

func (t ConvertCurrencyTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.TargetAmount); err != nil {
		return fmt.Errorf("target amount: %w", err)
	}

	if err := ledger.ValidateTransactionAmount(t.CostBasis); err != nil {
		return fmt.Errorf("cost basis: %w", err)
	}

	if err := ledger.ValidateCurrency(t.SourceCurrency); err != nil {
		return fmt.Errorf("source currency: %w", err)
	}

	if err := ledger.ValidateCurrency(t.TargetCurrency); err != nil {
		return fmt.Errorf("target currency: %w", err)
	}

	if t.CreditPriority != nil {
		if err := ledger.ValidateCreditPriority(*t.CreditPriority); err != nil {
			return fmt.Errorf("credit priority: %w", err)
		}
	}

	return nil
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
	priority := resolveCustomerFBOCreditPriority(t.CreditPriority)
	if t.CostBasis.IsNegative() {
		return nil, fmt.Errorf("failed to normalize cost basis: cost basis must be non-negative")
	}
	costBasis := t.CostBasis
	var targetSource *currencyx.Code
	if t.SourceCurrency.IsKnownFiat() && !t.TargetCurrency.IsKnownFiat() {
		targetSource = &t.SourceCurrency
	}

	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	sourceAccount, err := customerAccounts.FBOAccount.GetSubAccountForRoute(ctx, ledger.CustomerFBORouteParams{
		Currency:       t.SourceCurrency,
		CostBasis:      &costBasis,
		CreditPriority: priority,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get source sub-account: %w", err)
	}

	targetAccount, err := customerAccounts.FBOAccount.GetSubAccountForRoute(ctx, ledger.CustomerFBORouteParams{
		Currency:       t.TargetCurrency,
		Source:         targetSource,
		CostBasis:      &costBasis,
		CreditPriority: priority,
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
		Currency:  t.TargetCurrency,
		Source:    targetSource,
		CostBasis: &costBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brokerage target sub-account: %w", err)
	}

	sourceAmount := t.TargetAmount.Mul(t.CostBasis)

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			// Source currency
			{
				address: sourceAccount.Address(),
				amount:  sourceAmount.Neg(),
			},
			{
				address: brokerageSource.Address(),
				amount:  sourceAmount,
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

type ConvertCustomerReceivableCurrencyTemplate struct {
	At           time.Time
	SourceAmount alpacadecimal.Decimal
	CostBasis    alpacadecimal.Decimal

	SourceCurrency currencyx.Code
	TargetCurrency currencyx.Code

	Features       []string
	SourceChargeID *string
	SpendChargeID  *string
}

func (t ConvertCustomerReceivableCurrencyTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.SourceAmount); err != nil {
		return fmt.Errorf("source amount: %w", err)
	}

	if err := ledger.ValidateTransactionAmount(t.CostBasis); err != nil {
		return fmt.Errorf("cost basis: %w", err)
	}

	if err := ledger.ValidateCurrency(t.SourceCurrency); err != nil {
		return fmt.Errorf("source currency: %w", err)
	}

	if err := ledger.ValidateCurrency(t.TargetCurrency); err != nil {
		return fmt.Errorf("target currency: %w", err)
	}

	if t.SourceCurrency == t.TargetCurrency {
		return fmt.Errorf("source and target currency must differ")
	}

	if t.SourceCurrency.IsKnownFiat() {
		return fmt.Errorf("source currency must be custom")
	}

	if !t.TargetCurrency.IsKnownFiat() {
		return fmt.Errorf("target currency must be a known fiat currency")
	}

	return nil
}

var _ CustomerTransactionTemplate = (ConvertCustomerReceivableCurrencyTemplate{})

func (t ConvertCustomerReceivableCurrencyTemplate) correct(CorrectionInput) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(TemplateCode(t))
}

func (t ConvertCustomerReceivableCurrencyTemplate) typeGuard() guard {
	return true
}

func (t ConvertCustomerReceivableCurrencyTemplate) code() TransactionTemplateCode {
	return TemplateCodeConvertCustomerReceivableCurrency
}

func (t ConvertCustomerReceivableCurrencyTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	targetCalculator, err := t.TargetCurrency.Calculator()
	if err != nil {
		return nil, fmt.Errorf("creating target currency calculator: %w", err)
	}

	costBasis := t.CostBasis
	targetAmount := targetCalculator.RoundToPrecision(t.SourceAmount.Mul(costBasis))
	if err := ledger.ValidateTransactionAmount(targetAmount); err != nil {
		return nil, fmt.Errorf("target amount after rounding: %w", err)
	}
	source := &t.TargetCurrency

	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	sourceReceivable, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.SourceCurrency,
		Source:                         source,
		Features:                       t.Features,
		CostBasis:                      &costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get source receivable sub-account: %w", err)
	}

	targetReceivable, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.TargetCurrency,
		Features:                       t.Features,
		CostBasis:                      &costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get target receivable sub-account: %w", err)
	}

	businessAccounts, err := resolvers.AccountService.GetBusinessAccounts(ctx, customerID.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get business accounts: %w", err)
	}

	brokerageSource, err := businessAccounts.BrokerageAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
		Currency:  t.SourceCurrency,
		Source:    source,
		CostBasis: &costBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brokerage source sub-account: %w", err)
	}

	brokerageTarget, err := businessAccounts.BrokerageAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
		Currency:  t.TargetCurrency,
		CostBasis: &costBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brokerage target sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			{
				address: sourceReceivable.Address(),
				amount:  t.SourceAmount,
				identity: ledger.EntryIdentityParts{
					SourceChargeID: t.SourceChargeID,
					SpendChargeID:  t.SpendChargeID,
				},
			},
			{
				address: brokerageSource.Address(),
				amount:  t.SourceAmount.Neg(),
			},
			{
				address: targetReceivable.Address(),
				amount:  targetAmount.Neg(),
				identity: ledger.EntryIdentityParts{
					SourceChargeID: t.SourceChargeID,
					SpendChargeID:  t.SpendChargeID,
				},
			},
			{
				address: brokerageTarget.Address(),
				amount:  targetAmount,
			},
		},
	}, nil
}
