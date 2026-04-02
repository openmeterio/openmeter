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

func (t IssueCustomerReceivableTemplate) correct(context.Context, CorrectionInput, ResolverDependencies) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(templateName(t))
}

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

func (t FundCustomerReceivableTemplate) correct(context.Context, CorrectionInput, ResolverDependencies) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(templateName(t))
}

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

func (t SettleCustomerReceivablePaymentTemplate) correct(context.Context, CorrectionInput, ResolverDependencies) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(templateName(t))
}

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

// AttributeCustomerAdvanceReceivableCostBasisTemplate attributes existing open advance
// receivable (`cost_basis=nil`) into a known purchase cost-basis bucket.
type AttributeCustomerAdvanceReceivableCostBasisTemplate struct {
	At        time.Time
	Amount    alpacadecimal.Decimal
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
}

func (t AttributeCustomerAdvanceReceivableCostBasisTemplate) Validate() error {
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

func (t AttributeCustomerAdvanceReceivableCostBasisTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (AttributeCustomerAdvanceReceivableCostBasisTemplate{})

func (t AttributeCustomerAdvanceReceivableCostBasisTemplate) correct(context.Context, CorrectionInput, ResolverDependencies) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(templateName(t))
}

func (t AttributeCustomerAdvanceReceivableCostBasisTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	advanceReceivable, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.Currency,
		CostBasis:                      nil,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get advance receivable sub-account: %w", err)
	}

	attributedReceivable, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.Currency,
		CostBasis:                      t.CostBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get attributed receivable sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			{
				address: advanceReceivable.Address(),
				amount:  t.Amount,
			},
			{
				address: attributedReceivable.Address(),
				amount:  t.Amount.Neg(),
			},
		},
	}, nil
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

func (t CoverCustomerReceivableTemplate) correct(context.Context, CorrectionInput, ResolverDependencies) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(templateName(t))
}

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
