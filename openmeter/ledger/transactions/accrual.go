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

// TransferCustomerFBOToAccruedTemplate moves value from prioritized customer FBO
// sub-accounts into the customer's accrued account.
type TransferCustomerFBOToAccruedTemplate struct {
	At       time.Time
	Amount   alpacadecimal.Decimal
	Currency currencyx.Code
}

func (t TransferCustomerFBOToAccruedTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	return nil
}

func (t TransferCustomerFBOToAccruedTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (TransferCustomerFBOToAccruedTemplate{})

func (t TransferCustomerFBOToAccruedTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	collections, err := collectFromPrioritizedCustomerFBO(ctx, customerID, t.Currency, t.Amount, resolvers)
	if err != nil {
		return nil, fmt.Errorf("collect from prioritized FBO: %w", err)
	}
	if len(collections) == 0 {
		return nil, nil
	}

	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	accrued, err := customerAccounts.AccruedAccount.GetSubAccountForRoute(ctx, ledger.CustomerAccruedRouteParams{
		Currency: t.Currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get accrued sub-account: %w", err)
	}

	entryInputs := make([]*EntryInput, 0, len(collections)+1)
	totalCollected := alpacadecimal.Zero
	for _, collection := range collections {
		totalCollected = totalCollected.Add(collection.amount)
		entryInputs = append(entryInputs, &EntryInput{
			address: collection.subAccount.Address(),
			amount:  collection.amount.Neg(),
		})
	}

	entryInputs = append(entryInputs, &EntryInput{
		address: accrued.Address(),
		amount:  totalCollected,
	})

	return &TransactionInput{
		bookedAt:    t.At,
		entryInputs: entryInputs,
	}, nil
}

// TransferCustomerReceivableToAccruedTemplate acknowledges usage by moving it
// from receivable into the customer's accrued account.
type TransferCustomerReceivableToAccruedTemplate struct {
	At       time.Time
	Amount   alpacadecimal.Decimal
	Currency currencyx.Code
}

func (t TransferCustomerReceivableToAccruedTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	return nil
}

func (t TransferCustomerReceivableToAccruedTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (TransferCustomerReceivableToAccruedTemplate{})

func (t TransferCustomerReceivableToAccruedTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	receivable, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(ctx, ledger.CustomerReceivableRouteParams{
		Currency:                       t.Currency,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get receivable sub-account: %w", err)
	}

	accrued, err := customerAccounts.AccruedAccount.GetSubAccountForRoute(ctx, ledger.CustomerAccruedRouteParams{
		Currency: t.Currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get accrued sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			{
				address: receivable.Address(),
				amount:  t.Amount.Neg(),
			},
			{
				address: accrued.Address(),
				amount:  t.Amount,
			},
		},
	}, nil
}
