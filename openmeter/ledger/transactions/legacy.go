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

// legacyFundCustomerReceivableTemplate is an archived template kept for
// persisted ledger annotations. New payment flows should use
// SettleCustomerReceivableFromPaymentTemplate.
//
// Original semantics: funds the authorized receivable sub-account from wash.
type legacyFundCustomerReceivableTemplate struct {
	At        time.Time
	Amount    alpacadecimal.Decimal
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
}

func (t legacyFundCustomerReceivableTemplate) Validate() error {
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

var _ CustomerTransactionTemplate = (legacyFundCustomerReceivableTemplate{})

func (t legacyFundCustomerReceivableTemplate) correct(CorrectionInput) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(legacyTemplateNameFundCustomerReceivable)
}

func (t legacyFundCustomerReceivableTemplate) typeGuard() guard {
	return true
}

func (t legacyFundCustomerReceivableTemplate) code() TransactionTemplateCode {
	return ""
}

func (t legacyFundCustomerReceivableTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
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

// legacySettleCustomerReceivablePaymentTemplate is an archived template kept for
// persisted ledger annotations. New payment flows should use
// AuthorizeCustomerReceivablePaymentTemplate.
//
// Original semantics: moves authorized receivable staging into the open
// receivable route.
type legacySettleCustomerReceivablePaymentTemplate struct {
	At        time.Time
	Amount    alpacadecimal.Decimal
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
}

func (t legacySettleCustomerReceivablePaymentTemplate) Validate() error {
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

func (t legacySettleCustomerReceivablePaymentTemplate) typeGuard() guard {
	return true
}

func (t legacySettleCustomerReceivablePaymentTemplate) code() TransactionTemplateCode {
	return ""
}

var _ CustomerTransactionTemplate = (legacySettleCustomerReceivablePaymentTemplate{})

func (t legacySettleCustomerReceivablePaymentTemplate) correct(CorrectionInput) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(legacyTemplateNameSettleCustomerReceivablePayment)
}

func (t legacySettleCustomerReceivablePaymentTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
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
