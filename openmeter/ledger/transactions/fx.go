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
	// TaxCode  string // TBD
}

var _ CustomerTransactionTemplate = (ConvertCurrencyTemplate{})

func (t ConvertCurrencyTemplate) Resolve(ctx context.Context, customerID customer.CustomerID, resolvers Resolvers) (ledger.TransactionInput, error) {
	// Let's resolve the dimensions
	sourceCurrency, err := resolvers.DimensionService.GetCurrencyDimension(ctx, string(t.SourceCurrency))
	if err != nil {
		return nil, fmt.Errorf("failed to get source currency dimension: %w", err)
	}

	targetCurrency, err := resolvers.DimensionService.GetCurrencyDimension(ctx, string(t.TargetCurrency))
	if err != nil {
		return nil, fmt.Errorf("failed to get target currency dimension: %w", err)
	}

	// Let's fetch the customer accounts
	customerAccounts, err := resolvers.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer accounts: %w", err)
	}

	sourceAccount, err := customerAccounts.FBOAccount.GetSubAccountForDimensions(ctx, ledger.CustomerFBOSubAccountDimensions{
		Currency: sourceCurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get source sub-account: %w", err)
	}

	targetAccount, err := customerAccounts.FBOAccount.GetSubAccountForDimensions(ctx, ledger.CustomerFBOSubAccountDimensions{
		Currency: targetCurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get target sub-account: %w", err)
	}

	// Let's fetch the business accounts
	businessAccounts, err := resolvers.AccountService.GetBusinessAccounts(ctx, customerID.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get business accounts: %w", err)
	}

	brokerageSource, err := businessAccounts.BrokerageAccount.GetSubAccountForDimensions(ctx, ledger.BusinessSubAccountDimensions{
		Currency: sourceCurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brokerage source sub-account: %w", err)
	}

	brokerageTarget, err := businessAccounts.BrokerageAccount.GetSubAccountForDimensions(ctx, ledger.BusinessSubAccountDimensions{
		Currency: targetCurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brokerage target sub-account: %w", err)
	}

	sourceAmount := t.TargetAmount.Mul(t.CostBasis)

	// Now let's template the transaction
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
