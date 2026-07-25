package transactions

import (
	"cmp"
	"context"
	"fmt"
	"sort"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type postingAddressBalance struct {
	address  ledger.PostingAddress
	balance  alpacadecimal.Decimal
	identity ledger.EntryIdentityParts
}

type postingAddressAmount struct {
	address  ledger.PostingAddress
	amount   alpacadecimal.Decimal
	identity ledger.EntryIdentityParts
}

// PostingAmount is a preselected amount to post against an address.
type PostingAmount struct {
	Address     ledger.PostingAddress
	Amount      alpacadecimal.Decimal
	Identity    ledger.EntryIdentityParts
	Annotations models.Annotations
}

type accountIdentifier interface {
	ID() models.NamespacedID
}

func collectFromAttributableCustomerAccrued(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
	target alpacadecimal.Decimal,
	deps ResolverDependencies,
) ([]postingAddressAmount, error) {
	customerAccounts, err := deps.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	accruedAccountWithID, ok := customerAccounts.AccruedAccount.(accountIdentifier)
	if !ok {
		return nil, fmt.Errorf("customer accrued account does not expose an ID")
	}

	accruedAccountID := accruedAccountWithID.ID().ID
	buckets, err := deps.BalanceQuerier.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: customerID.Namespace,
		Filters: ledger.Filters{
			AccountID: &accruedAccountID,
			Route: ledger.RouteFilter{
				Currency: currency,
			},
		},
		GroupBy: []string{
			ledger.BalanceBucketGroupBySourceChargeID,
			ledger.BalanceBucketGroupBySpendChargeID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list attributable accrued buckets: %w", err)
	}

	sources := make([]postingAddressBalance, 0, len(buckets))
	for _, bucket := range buckets {
		route := bucket.Address.Route().Route()
		if route.Currency != currency || route.CostBasis == nil {
			continue
		}

		sources = append(sources, postingAddressBalance{
			address: bucket.Address,
			balance: bucket.SettledAmount,
			identity: ledger.EntryIdentityParts{
				SourceChargeID: bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID],
				SpendChargeID:  bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID],
			},
		})
	}

	// Recognition correction sorts the original accrued source entries by
	// sub-account id and unwinds from the end. Keep forward recognition ordered
	// the same way so partial corrections are deterministic.
	//
	// There is no business requirement on the priority order of earning recognition.
	sort.Slice(sources, func(i, j int) bool {
		if c := cmp.Compare(sources[i].address.SubAccountID(), sources[j].address.SubAccountID()); c != 0 {
			return c < 0
		}

		leftIdentity, _ := sources[i].identity.Text()
		rightIdentity, _ := sources[j].identity.Text()
		return cmp.Compare(string(leftIdentity), string(rightIdentity)) < 0
	})

	return collectFromPostingAddressBalanceSources(sources, target), nil
}

func collectFromPostingAddressBalanceSources(sources []postingAddressBalance, target alpacadecimal.Decimal) []postingAddressAmount {
	remaining := target
	out := make([]postingAddressAmount, 0, len(sources))

	for _, source := range sources {
		if !remaining.IsPositive() {
			break
		}

		if !source.balance.IsPositive() {
			continue
		}

		amount := source.balance
		if source.balance.GreaterThan(remaining) {
			amount = remaining
		}

		out = append(out, postingAddressAmount{
			address:  source.address,
			amount:   amount,
			identity: source.identity,
		})
		remaining = remaining.Sub(amount)
	}

	return out
}

func decimalPointersEqual(left, right *alpacadecimal.Decimal) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.Equal(*right)
	}
}

func settledBalanceForSubAccount(ctx context.Context, deps ResolverDependencies, subAccount ledger.SubAccount) (alpacadecimal.Decimal, error) {
	balance, err := deps.BalanceQuerier.GetSubAccountBalance(ctx, subAccount, ledger.BalanceQuery{})
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get balance for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return balance, nil
}
