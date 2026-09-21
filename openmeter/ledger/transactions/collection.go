package transactions

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
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

type collectFromAttributableCustomerAccruedInput struct {
	CustomerID customer.CustomerID
	Currency   currencies.CurrencyReference
	// Target caps the amount selected from eligible accrued balances.
	Target alpacadecimal.Decimal
	// OriginTracked selects entries with a collection origin; false selects legacy entries without one.
	OriginTracked bool
	// AsOf is the accounting-time boundary for the accrued balances being selected.
	AsOf time.Time
}

func (i collectFromAttributableCustomerAccruedInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if err := ledger.ValidateTransactionAmount(i.Target); err != nil {
		errs = append(errs, fmt.Errorf("target: %w", err))
	}

	if i.AsOf.IsZero() {
		errs = append(errs, errors.New("as of is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func collectFromAttributableCustomerAccrued(
	ctx context.Context,
	deps ResolverDependencies,
	input collectFromAttributableCustomerAccruedInput,
) ([]postingAddressAmount, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	customerAccounts, err := deps.AccountService.GetCustomerAccounts(ctx, input.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	accruedAccountWithID, ok := customerAccounts.AccruedAccount.(accountIdentifier)
	if !ok {
		return nil, fmt.Errorf("customer accrued account does not expose an ID")
	}

	accruedAccountID := accruedAccountWithID.ID().ID
	buckets, err := deps.BalanceQuerier.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: input.CustomerID.Namespace,
		Filters: ledger.Filters{
			AccountID: &accruedAccountID,
			AsOf:      &input.AsOf,
			Route: ledger.RouteFilter{
				Currency: input.Currency,
			},
		},
		GroupBy: []string{
			ledger.BalanceBucketGroupBySourceChargeID,
			ledger.BalanceBucketGroupBySpendChargeID,
			ledger.BalanceBucketGroupByCollectionOriginID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list attributable accrued buckets: %w", err)
	}

	sources := make([]postingAddressBalance, 0, len(buckets))
	for _, bucket := range buckets {
		route := bucket.Address.Route().Route()
		if !route.Currency.Equal(input.Currency) || route.CostBasis == nil {
			continue
		}

		identity := ledger.EntryIdentityParts{
			Provenance: ledger.Provenance{
				SourceChargeID:     bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID],
				CollectionOriginID: bucket.GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID],
				SpendChargeID:      bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID],
			},
		}

		hasCollectionOrigin := identity.CollectionOriginID != nil
		if hasCollectionOrigin != input.OriginTracked {
			continue
		}

		if !isCreditBackedAccruedIdentity(identity) {
			continue
		}

		sources = append(sources, postingAddressBalance{
			address:  bucket.Address,
			balance:  bucket.SettledAmount,
			identity: identity,
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

	return collectFromPostingAddressBalanceSources(sources, input.Target), nil
}

// isCreditBackedAccruedIdentity reports whether accrued value has the distinct
// source-credit and spend-charge provenance required for credit-backed earnings
// recognition. Value without this provenance can be invoice-backed or an
// unbackfilled advance.
func isCreditBackedAccruedIdentity(identity ledger.EntryIdentityParts) bool {
	return identity.SourceChargeID != nil &&
		identity.SpendChargeID != nil &&
		*identity.SourceChargeID != *identity.SpendChargeID
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
