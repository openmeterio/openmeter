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

// IssueCustomerReceivableTemplate is a transaction increasing the customer's balance against an outstanding receivable account
type IssueCustomerReceivableTemplate struct {
	At        time.Time
	Amount    alpacadecimal.Decimal
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
	// Optional, defaults to ledger.DefaultCustomerFBOPriority.
	CreditPriority *int
}

func (t IssueCustomerReceivableTemplate) Validate() error {
	if t.Amount.IsNegative() {
		return fmt.Errorf("amount must be positive")
	}

	if t.Amount.IsZero() {
		return fmt.Errorf("amount must be non-zero")
	}

	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if t.CostBasis != nil {
		if err := ledger.ValidateCostBasis(*t.CostBasis); err != nil {
			return fmt.Errorf("cost basis: %w", err)
		}
	}

	if t.CreditPriority != nil {
		if err := ledger.ValidateCreditPriority(*t.CreditPriority); err != nil {
			return fmt.Errorf("credit priority: %w", err)
		}
	}

	return nil
}

func (t IssueCustomerReceivableTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (IssueCustomerReceivableTemplate{})

func (t IssueCustomerReceivableTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	priority := resolveCustomerFBOCreditPriority(t.CreditPriority)

	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	fbo, err := customerAccounts.FBOAccount.GetSubAccountForRoute(ctx, ledger.CustomerFBORouteParams{
		Currency:       t.Currency,
		CostBasis:      t.CostBasis,
		CreditPriority: priority,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get FBO sub-account: %w", err)
	}

	rec, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.Currency,
		CostBasis:                      t.CostBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get receivable sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			{
				address: fbo.Address(),
				amount:  t.Amount,
			},
			{
				address: rec.Address(),
				amount:  t.Amount.Neg(),
			},
		},
	}, nil
}

// FundCustomerReceivableTemplate funds the authorized receivable sub-account from wash.
type FundCustomerReceivableTemplate struct {
	At        time.Time
	Amount    alpacadecimal.Decimal
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
}

func (t FundCustomerReceivableTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if t.CostBasis != nil {
		if err := ledger.ValidateCostBasis(*t.CostBasis); err != nil {
			return fmt.Errorf("cost basis: %w", err)
		}
	}

	return nil
}

var _ CustomerTransactionTemplate = (FundCustomerReceivableTemplate{})

func (t FundCustomerReceivableTemplate) typeGuard() guard {
	return true
}

func (t FundCustomerReceivableTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	rec, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.Currency,
		CostBasis:                      t.CostBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusAuthorized,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get receivable sub-account: %w", err)
	}

	businessAccounts, err := resolvers.AccountService.GetBusinessAccounts(ctx, customerID.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get business accounts: %w", err)
	}

	wash, err := businessAccounts.WashAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
		Currency:  t.Currency,
		CostBasis: t.CostBasis,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get wash sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			{
				address: wash.Address(),
				amount:  t.Amount.Neg(),
			},
			{
				address: rec.Address(),
				amount:  t.Amount,
			},
		},
	}, nil
}

// SettleCustomerReceivablePaymentTemplate moves authorized receivable staging into the open receivable route.
type SettleCustomerReceivablePaymentTemplate struct {
	At        time.Time
	Amount    alpacadecimal.Decimal
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
}

func (t SettleCustomerReceivablePaymentTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if t.CostBasis != nil {
		if err := ledger.ValidateCostBasis(*t.CostBasis); err != nil {
			return fmt.Errorf("cost basis: %w", err)
		}
	}

	return nil
}

func (t SettleCustomerReceivablePaymentTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (SettleCustomerReceivablePaymentTemplate{})

func (t SettleCustomerReceivablePaymentTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	authorizedReceivable, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.Currency,
		CostBasis:                      t.CostBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusAuthorized,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get authorized receivable sub-account: %w", err)
	}

	openReceivable, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.Currency,
		CostBasis:                      t.CostBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get receivable sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			{
				address: authorizedReceivable.Address(),
				amount:  t.Amount.Neg(),
			},
			{
				address: openReceivable.Address(),
				amount:  t.Amount,
			},
		},
	}, nil
}

type customerSubAccountResolver func(ctx context.Context, customerAccounts ledger.CustomerAccounts, costBasis *alpacadecimal.Decimal) (ledger.SubAccount, error)

func resolveCustomerCostBasisTranslation(
	ctx context.Context,
	customerID customer.CustomerID,
	resolvers ResolverDependencies,
	at time.Time,
	amount alpacadecimal.Decimal,
	fromCostBasis *alpacadecimal.Decimal,
	toCostBasis *alpacadecimal.Decimal,
	resolveSubAccount customerSubAccountResolver,
) (ledger.TransactionInput, error) {
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	fromSubAccount, err := resolveSubAccount(ctx, customerAccounts, fromCostBasis)
	if err != nil {
		return nil, err
	}

	toSubAccount, err := resolveSubAccount(ctx, customerAccounts, toCostBasis)
	if err != nil {
		return nil, err
	}

	return &TransactionInput{
		bookedAt: at,
		entryInputs: []*EntryInput{
			{
				address: fromSubAccount.Address(),
				amount:  amount.Neg(),
			},
			{
				address: toSubAccount.Address(),
				amount:  amount,
			},
		},
	}, nil
}

// FundCustomerAdvanceReceivableTemplate applies known receivable funding against the
// open advance receivable bucket (`cost_basis=nil`) without changing authorization stage.
type FundCustomerAdvanceReceivableTemplate struct {
	At        time.Time
	Amount    alpacadecimal.Decimal
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
}

func (t FundCustomerAdvanceReceivableTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if t.CostBasis == nil {
		return fmt.Errorf("cost basis is required")
	}

	if err := ledger.ValidateCostBasis(*t.CostBasis); err != nil {
		return fmt.Errorf("cost basis: %w", err)
	}

	return nil
}

func (t FundCustomerAdvanceReceivableTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (FundCustomerAdvanceReceivableTemplate{})

func (t FundCustomerAdvanceReceivableTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	return resolveCustomerCostBasisTranslation(
		ctx,
		customerID,
		resolvers,
		t.At,
		t.Amount,
		t.CostBasis,
		nil,
		func(ctx context.Context, customerAccounts ledger.CustomerAccounts, costBasis *alpacadecimal.Decimal) (ledger.SubAccount, error) {
			subAccount, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
				Currency:                       t.Currency,
				CostBasis:                      costBasis,
				TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get receivable sub-account: %w", err)
			}

			return subAccount, nil
		},
	)
}

// CoverCustomerReceivableTemplate covers a customer receivable account from FBO account
type CoverCustomerReceivableTemplate struct {
	At        time.Time
	Amount    alpacadecimal.Decimal
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
	// Optional, defaults to 100.
	CreditPriority *int
}

func (t CoverCustomerReceivableTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if t.CostBasis != nil {
		if err := ledger.ValidateCostBasis(*t.CostBasis); err != nil {
			return fmt.Errorf("cost basis: %w", err)
		}
	}

	if t.CreditPriority != nil {
		if err := ledger.ValidateCreditPriority(*t.CreditPriority); err != nil {
			return fmt.Errorf("credit priority: %w", err)
		}
	}

	return nil
}

func (t CoverCustomerReceivableTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (CoverCustomerReceivableTemplate{})

func (t CoverCustomerReceivableTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	priority := resolveCustomerFBOCreditPriority(t.CreditPriority)

	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	fbo, err := customerAccounts.FBOAccount.GetSubAccountForRoute(ctx, ledger.CustomerFBORouteParams{
		Currency:       t.Currency,
		CostBasis:      t.CostBasis,
		CreditPriority: priority,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get FBO sub-account: %w", err)
	}

	rec, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.Currency,
		CostBasis:                      t.CostBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get receivable sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			{
				address: fbo.Address(),
				amount:  t.Amount.Neg(),
			},
			{
				address: rec.Address(),
				amount:  t.Amount,
			},
		},
	}, nil
}

// TranslateCustomerFBOCostBasisTemplate moves customer FBO value between cost-basis buckets
// while keeping the same priority bucket.
type TranslateCustomerFBOCostBasisTemplate struct {
	At             time.Time
	Amount         alpacadecimal.Decimal
	Currency       currencyx.Code
	FromCostBasis  *alpacadecimal.Decimal
	ToCostBasis    *alpacadecimal.Decimal
	CreditPriority *int
}

func (t TranslateCustomerFBOCostBasisTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if t.FromCostBasis == nil {
		return fmt.Errorf("from cost basis is required")
	}

	if err := ledger.ValidateCostBasis(*t.FromCostBasis); err != nil {
		return fmt.Errorf("from cost basis: %w", err)
	}

	if t.ToCostBasis == nil {
		return fmt.Errorf("to cost basis is required")
	}

	if err := ledger.ValidateCostBasis(*t.ToCostBasis); err != nil {
		return fmt.Errorf("to cost basis: %w", err)
	}

	if decimalPointersEqual(t.FromCostBasis, t.ToCostBasis) {
		return fmt.Errorf("from and to cost basis must differ")
	}

	if t.CreditPriority != nil {
		if err := ledger.ValidateCreditPriority(*t.CreditPriority); err != nil {
			return fmt.Errorf("credit priority: %w", err)
		}
	}

	return nil
}

func (t TranslateCustomerFBOCostBasisTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (TranslateCustomerFBOCostBasisTemplate{})

func (t TranslateCustomerFBOCostBasisTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	priority := resolveCustomerFBOCreditPriority(t.CreditPriority)

	return resolveCustomerCostBasisTranslation(
		ctx,
		customerID,
		resolvers,
		t.At,
		t.Amount,
		t.FromCostBasis,
		t.ToCostBasis,
		func(ctx context.Context, customerAccounts ledger.CustomerAccounts, costBasis *alpacadecimal.Decimal) (ledger.SubAccount, error) {
			subAccount, err := customerAccounts.FBOAccount.GetSubAccountForRoute(ctx, ledger.CustomerFBORouteParams{
				Currency:       t.Currency,
				CostBasis:      costBasis,
				CreditPriority: priority,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get FBO sub-account: %w", err)
			}

			return subAccount, nil
		},
	)
}
