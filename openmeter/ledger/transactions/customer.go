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

func (t IssueCustomerReceivableTemplate) correct(scope CorrectionInput) ([]ledger.TransactionInput, error) {
	var fboAddress ledger.PostingAddress
	var receivableAddress ledger.PostingAddress
	var fboAmount alpacadecimal.Decimal
	var receivableAmount alpacadecimal.Decimal

	for _, entry := range scope.OriginalTransaction.Entries() {
		switch {
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO && entry.Amount().IsPositive():
			fboAddress = entry.PostingAddress()
			fboAmount = fboAmount.Add(entry.Amount())
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerReceivable && entry.Amount().IsNegative():
			receivableAddress = entry.PostingAddress()
			receivableAmount = receivableAmount.Add(entry.Amount().Abs())
		}
	}

	if fboAddress == nil || receivableAddress == nil {
		return nil, fmt.Errorf("issue receivable correction requires original FBO and receivable entries")
	}

	if scope.Amount.GreaterThan(fboAmount) || scope.Amount.GreaterThan(receivableAmount) {
		return nil, fmt.Errorf("issue receivable correction amount %s exceeds original transaction amount", scope.Amount.String())
	}

	return []ledger.TransactionInput{
		&TransactionInput{
			bookedAt: scope.At,
			entryInputs: []*EntryInput{
				{
					address: fboAddress,
					amount:  scope.Amount.Neg(),
				},
				{
					address: receivableAddress,
					amount:  scope.Amount,
				},
			},
		},
	}, nil
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

// SettleCustomerReceivableFromPaymentTemplate records settled payment funds by
// clearing authorized receivable from wash.
type SettleCustomerReceivableFromPaymentTemplate struct {
	At        time.Time
	Amount    alpacadecimal.Decimal
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
}

func (t SettleCustomerReceivableFromPaymentTemplate) Validate() error {
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

var _ CustomerTransactionTemplate = (SettleCustomerReceivableFromPaymentTemplate{})

func (t SettleCustomerReceivableFromPaymentTemplate) correct(CorrectionInput) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(TemplateCode(t))
}

func (t SettleCustomerReceivableFromPaymentTemplate) typeGuard() guard {
	return true
}

func (t SettleCustomerReceivableFromPaymentTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
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

// AuthorizeCustomerReceivablePaymentTemplate moves open receivable into the
// authorized receivable route without moving funds across the external cash boundary.
type AuthorizeCustomerReceivablePaymentTemplate struct {
	At        time.Time
	Amount    alpacadecimal.Decimal
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
}

func (t AuthorizeCustomerReceivablePaymentTemplate) Validate() error {
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

func (t AuthorizeCustomerReceivablePaymentTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (AuthorizeCustomerReceivablePaymentTemplate{})

func (t AuthorizeCustomerReceivablePaymentTemplate) correct(CorrectionInput) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(TemplateCode(t))
}

func (t AuthorizeCustomerReceivablePaymentTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
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

func (t AttributeCustomerAdvanceReceivableCostBasisTemplate) correct(scope CorrectionInput) ([]ledger.TransactionInput, error) {
	var advanceReceivableAddress ledger.PostingAddress
	var attributedReceivableAddress ledger.PostingAddress
	var advanceReceivableAmount alpacadecimal.Decimal
	var attributedReceivableAmount alpacadecimal.Decimal

	for _, entry := range scope.OriginalTransaction.Entries() {
		switch {
		case entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerReceivable:
			continue
		case entry.Amount().IsPositive():
			advanceReceivableAddress = entry.PostingAddress()
			advanceReceivableAmount = advanceReceivableAmount.Add(entry.Amount())
		case entry.Amount().IsNegative():
			attributedReceivableAddress = entry.PostingAddress()
			attributedReceivableAmount = attributedReceivableAmount.Add(entry.Amount().Abs())
		}
	}

	if advanceReceivableAddress == nil || attributedReceivableAddress == nil {
		return nil, fmt.Errorf("advance receivable attribution correction requires original receivable entries")
	}

	if scope.Amount.GreaterThan(advanceReceivableAmount) || scope.Amount.GreaterThan(attributedReceivableAmount) {
		return nil, fmt.Errorf("advance receivable attribution correction amount %s exceeds original transaction amount", scope.Amount.String())
	}

	return []ledger.TransactionInput{
		&TransactionInput{
			bookedAt: scope.At,
			entryInputs: []*EntryInput{
				{
					address: advanceReceivableAddress,
					amount:  scope.Amount.Neg(),
				},
				{
					address: attributedReceivableAddress,
					amount:  scope.Amount,
				},
			},
		},
	}, nil
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

func (t CoverCustomerReceivableTemplate) correct(CorrectionInput) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(TemplateCode(t))
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
