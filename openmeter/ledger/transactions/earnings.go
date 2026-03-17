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

// RecognizeEarningsFromAccruedTemplate recognizes earnings from invoiced values
type RecognizeEarningsFromAccruedTemplate struct {
	At       time.Time
	Amount   alpacadecimal.Decimal
	Currency currencyx.Code
}

func (t RecognizeEarningsFromAccruedTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (RecognizeEarningsFromAccruedTemplate{})

func (t RecognizeEarningsFromAccruedTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
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

	businessAccounts, err := resolvers.AccountService.GetBusinessAccounts(ctx, customerID.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get business accounts: %w", err)
	}

	earnings, err := businessAccounts.EarningsAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
		Currency: t.Currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get earnings sub-account: %w", err)
	}

	return &TransactionInput{
		bookedAt: t.At,
		entryInputs: []*EntryInput{
			{
				address: accrued.Address(),
				amount:  t.Amount.Neg(),
			},
			{
				address: earnings.Address(),
				amount:  t.Amount,
			},
		},
	}, nil
}
