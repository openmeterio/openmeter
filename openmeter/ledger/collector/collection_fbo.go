package collector

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/cmpx"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type fboCollectionSource struct {
	address        ledger.PostingAddress
	available      alpacadecimal.Decimal
	creditPriority int
	expiresAt      *time.Time
	cursor         string
	breakagePlan   *breakage.Plan
}

type fboCollectionSelection struct {
	source fboCollectionSource
	amount alpacadecimal.Decimal
}

type fboCollectionSelections []fboCollectionSelection

var _ cmpx.Comparable[fboCollectionSource] = fboCollectionSource{}

type accountIdentifier interface {
	ID() models.NamespacedID
}

func (c *accrualCollector) collectCustomerFBO(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
	target alpacadecimal.Decimal,
	asOf time.Time,
) ([]transactions.PostingAmount, error) {
	selections, err := c.collectCustomerFBOSelections(ctx, customerID, currency, target, asOf)
	if err != nil {
		return nil, err
	}

	return fboCollectionSelections(selections).postingAmounts(), nil
}

func (c *accrualCollector) collectCustomerFBOSelections(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
	target alpacadecimal.Decimal,
	asOf time.Time,
) ([]fboCollectionSelection, error) {
	sources, err := c.listCustomerFBOSources(ctx, customerID, currency, asOf)
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(sources, cmpx.Compare[fboCollectionSource])

	return selectFBOSources(sources, target), nil
}

func (s fboCollectionSource) Compare(other fboCollectionSource) int {
	if c := cmp.Compare(s.creditPriority, other.creditPriority); c != 0 {
		return c
	}

	if c := compareOptionalTime(s.expiresAt, other.expiresAt); c != 0 {
		return c
	}

	return cmp.Compare(s.cursor, other.cursor)
}

func compareOptionalTime(left, right *time.Time) int {
	switch {
	case left == nil && right == nil:
		return 0
	case left == nil:
		return 1
	case right == nil:
		return -1
	default:
		return left.Compare(*right)
	}
}

func (c *accrualCollector) listCustomerFBOSources(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
	asOf time.Time,
) ([]fboCollectionSource, error) {
	if c.deps.AccountCatalog == nil {
		return nil, fmt.Errorf("account catalog is required")
	}

	if c.deps.BalanceQuerier == nil {
		return nil, fmt.Errorf("balance querier is required")
	}

	customerAccounts, err := c.deps.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	fboAccountWithID, ok := customerAccounts.FBOAccount.(accountIdentifier)
	if !ok {
		return nil, fmt.Errorf("customer FBO account does not expose an ID")
	}

	subAccounts, err := c.deps.AccountCatalog.ListSubAccounts(ctx, ledger.ListSubAccountsInput{
		Namespace: fboAccountWithID.ID().Namespace,
		AccountID: fboAccountWithID.ID().ID,
	})
	if err != nil {
		return nil, fmt.Errorf("list sub-accounts: %w", err)
	}

	sources := make([]fboCollectionSource, 0, len(subAccounts))
	sourcesBySubAccountID := make(map[string]fboCollectionSource, len(subAccounts))
	availableBySubAccountID := make(map[string]alpacadecimal.Decimal, len(subAccounts))
	for _, subAccount := range subAccounts {
		route := subAccount.Route()
		if route.Currency != currency {
			continue
		}

		balance, err := c.settledSubAccountBalance(ctx, subAccount, asOf)
		if err != nil {
			return nil, err
		}

		source := fboCollectionSource{
			address:        subAccount.Address(),
			available:      balance,
			creditPriority: customerFBOPriority(route),
			cursor:         subAccount.Address().SubAccountID(),
		}
		sourcesBySubAccountID[subAccount.Address().SubAccountID()] = source
		availableBySubAccountID[subAccount.Address().SubAccountID()] = balance
		sources = append(sources, source)
	}

	if c.breakage == nil {
		return sources, nil
	}

	openPlans, err := c.breakage.ListPlans(ctx, breakage.ListPlansInput{
		CustomerID: customerID,
		Currency:   currency,
		AsOf:       asOf,
	})
	if err != nil {
		return nil, fmt.Errorf("list open breakage plans: %w", err)
	}

	breakageSources := make([]fboCollectionSource, 0, len(openPlans)+len(subAccounts))
	for _, plan := range openPlans {
		remainingBalance := availableBySubAccountID[plan.FBOSubAccountID]
		if !remainingBalance.IsPositive() {
			continue
		}

		available := plan.OpenAmount
		if available.GreaterThan(remainingBalance) {
			available = remainingBalance
		}
		availableBySubAccountID[plan.FBOSubAccountID] = remainingBalance.Sub(available)

		if !available.IsPositive() {
			continue
		}

		planCopy := plan
		expiresAt := plan.ExpiresAt
		breakageSources = append(breakageSources, fboCollectionSource{
			address:        plan.FBOAddress,
			available:      available,
			creditPriority: plan.CreditPriority,
			expiresAt:      &expiresAt,
			cursor:         plan.ID.ID,
			breakagePlan:   &planCopy,
		})
	}

	for subAccountID, source := range sourcesBySubAccountID {
		source.available = availableBySubAccountID[subAccountID]
		if !source.available.IsPositive() {
			continue
		}

		breakageSources = append(breakageSources, source)
	}

	return breakageSources, nil
}

func (s fboCollectionSelections) postingAmounts() []transactions.PostingAmount {
	out := make([]transactions.PostingAmount, 0, len(s))
	indexBySubAccountID := make(map[string]int, len(s))

	for _, selection := range s {
		subAccountID := selection.source.address.SubAccountID()
		if idx, ok := indexBySubAccountID[subAccountID]; ok {
			out[idx].Amount = out[idx].Amount.Add(selection.amount)
			continue
		}

		indexBySubAccountID[subAccountID] = len(out)
		out = append(out, transactions.PostingAmount{
			Address: selection.source.address,
			Amount:  selection.amount,
		})
	}

	return out
}

func selectFBOSources(sources []fboCollectionSource, target alpacadecimal.Decimal) []fboCollectionSelection {
	remaining := target
	out := make([]fboCollectionSelection, 0, len(sources))

	for _, source := range sources {
		if !remaining.IsPositive() {
			break
		}

		if !source.available.IsPositive() {
			continue
		}

		amount := source.available
		if source.available.GreaterThan(remaining) {
			amount = remaining
		}

		out = append(out, fboCollectionSelection{
			source: source,
			amount: amount,
		})
		remaining = remaining.Sub(amount)
	}

	return out
}

func selectFBOPostingAmounts(sources []fboCollectionSource, target alpacadecimal.Decimal) []transactions.PostingAmount {
	return fboCollectionSelections(selectFBOSources(sources, target)).postingAmounts()
}

func customerFBOPriority(route ledger.Route) int {
	if route.CreditPriority == nil {
		return ledger.DefaultCustomerFBOPriority
	}

	return *route.CreditPriority
}

func (c *accrualCollector) settledSubAccountBalance(ctx context.Context, subAccount ledger.SubAccount, asOf time.Time) (alpacadecimal.Decimal, error) {
	balance, err := c.deps.BalanceQuerier.GetSubAccountBalance(ctx, subAccount, ledger.BalanceQuery{
		AsOf: &asOf,
	})
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get balance for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return balance.Settled(), nil
}
