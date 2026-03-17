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

var _ CustomerTransactionTemplate = (ConvertCurrencyTemplate{})

func (t ConvertCurrencyTemplate) typeGuard() guard {
	return true
}

func (t ConvertCurrencyTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	priority := resolveCustomerFBOCreditPriority(t.CreditPriority)

	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	sourceAccount, err := customerAccounts.FBOAccount.GetSubAccountForRoute(ctx, ledger.CustomerFBORouteParams{
		Currency:       string(t.SourceCurrency),
		CreditPriority: priority,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get source sub-account: %w", err)
	}

	targetAccount, err := customerAccounts.FBOAccount.GetSubAccountForRoute(ctx, ledger.CustomerFBORouteParams{
		Currency:       string(t.TargetCurrency),
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
		Currency: string(t.SourceCurrency),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brokerage source sub-account: %w", err)
	}

	brokerageTarget, err := businessAccounts.BrokerageAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
		Currency: string(t.TargetCurrency),
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
