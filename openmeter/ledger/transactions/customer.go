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

// A Customer Receivable is a transaction increasing the customer's balance against an outstanding receivable account
type IssueCustomerReceivableTemplate struct {
	At       time.Time
	Amount   alpacadecimal.Decimal
	Currency currencyx.Code
	// TaxCode  string // TBD
}

var _ CustomerTransactionTemplate = (IssueCustomerReceivableTemplate{})

func (t IssueCustomerReceivableTemplate) Resolve(ctx context.Context, customerID customer.CustomerID, resolvers Resolvers) (ledger.TransactionInput, error) {
	// Let's resolve the dimensions
	currency, err := resolvers.DimensionService.GetCurrencyDimension(ctx, string(t.Currency))
	if err != nil {
		return nil, fmt.Errorf("failed to get currency dimension: %w", err)
	}

	// Let's fetch the customer accounts
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	fbo, err := customerAccounts.FBOAccount.GetSubAccountForDimensions(ctx, ledger.CustomerFBOSubAccountDimensions{
		Currency: currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get FBO sub-account: %w", err)
	}

	rec, err := customerAccounts.ReceivableAccount.GetSubAccountForDimensions(ctx, ledger.CustomerReceivableSubAccountDimensions{
		Currency: currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get receivable sub-account: %w", err)
	}

	// Now let's template the transaction
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

// Funds a customer receivable account from wash account
type FundCustomerReceivableTemplate struct {
	At       time.Time
	Amount   alpacadecimal.Decimal
	Currency currencyx.Code
	// TaxCode  string // TBD
}

var _ CustomerTransactionTemplate = (FundCustomerReceivableTemplate{})

func (t FundCustomerReceivableTemplate) Resolve(ctx context.Context, customerID customer.CustomerID, resolvers Resolvers) (ledger.TransactionInput, error) {
	// Let's resolve the dimensions
	currency, err := resolvers.DimensionService.GetCurrencyDimension(ctx, string(t.Currency))
	if err != nil {
		return nil, fmt.Errorf("failed to get currency dimension: %w", err)
	}

	// Let's fetch the customer accounts
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	rec, err := customerAccounts.ReceivableAccount.GetSubAccountForDimensions(ctx, ledger.CustomerReceivableSubAccountDimensions{
		Currency: currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get receivable sub-account: %w", err)
	}

	businessAccounts, err := resolvers.AccountService.GetBusinessAccounts(ctx, customerID.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get business accounts: %w", err)
	}

	wash, err := businessAccounts.WashAccount.GetSubAccountForDimensions(ctx, ledger.BusinessSubAccountDimensions{
		Currency: currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get wash sub-account: %w", err)
	}

	// Now let's template the transaction
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

// Covers a customer receivable account from FBO account
type CoverCustomerReceivableTemplate struct {
	At       time.Time
	Amount   alpacadecimal.Decimal
	Currency currencyx.Code
	// TaxCode  string // TBD
}

var _ CustomerTransactionTemplate = (CoverCustomerReceivableTemplate{})

func (t CoverCustomerReceivableTemplate) Resolve(ctx context.Context, customerID customer.CustomerID, resolvers Resolvers) (ledger.TransactionInput, error) {
	// Let's resolve the dimensions
	currency, err := resolvers.DimensionService.GetCurrencyDimension(ctx, string(t.Currency))
	if err != nil {
		return nil, fmt.Errorf("failed to get currency dimension: %w", err)
	}

	// Let's fetch the customer accounts
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	fbo, err := customerAccounts.FBOAccount.GetSubAccountForDimensions(ctx, ledger.CustomerFBOSubAccountDimensions{
		Currency: currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get FBO sub-account: %w", err)
	}

	rec, err := customerAccounts.ReceivableAccount.GetSubAccountForDimensions(ctx, ledger.CustomerReceivableSubAccountDimensions{
		Currency: currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get receivable sub-account: %w", err)
	}

	// Now let's template the transaction
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

// Recognizes earnings from a customer's balance account
type RecognizeEarningsFromCreditsTemplate struct {
	At       time.Time
	Amount   alpacadecimal.Decimal
	Currency currencyx.Code
	// TaxCode  string // TBD
}

var _ CustomerTransactionTemplate = (RecognizeEarningsFromCreditsTemplate{})

func (t RecognizeEarningsFromCreditsTemplate) Resolve(ctx context.Context, customerID customer.CustomerID, resolvers Resolvers) (ledger.TransactionInput, error) {
	panic("not implemented")
}

// Recognizes earnings from invoiced values
type RecognizeEarningsFromAccruedTemplate struct {
	At       time.Time
	Amount   alpacadecimal.Decimal
	Currency currencyx.Code
	// TaxCode  string // TBD
}

var _ CustomerTransactionTemplate = (RecognizeEarningsFromAccruedTemplate{})

func (t RecognizeEarningsFromAccruedTemplate) Resolve(ctx context.Context, customerID customer.CustomerID, resolvers Resolvers) (ledger.TransactionInput, error) {
	panic("not implemented")
}
