package collector

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/cmpx"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type fboCollectionSource struct {
	address           ledger.PostingAddress
	sourceChargeID    *string
	available         alpacadecimal.Decimal
	creditPriority    int
	featureRestricted bool
	expiresAt         *time.Time
	cursor            string
	breakagePlan      *breakage.Plan
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
	featureKey string,
	target alpacadecimal.Decimal,
	asOf time.Time,
) ([]transactions.PostingAmount, error) {
	selections, err := c.collectCustomerFBOSelections(ctx, customerID, currency, featureKey, target, asOf)
	if err != nil {
		return nil, err
	}

	return fboCollectionSelections(selections).postingAmounts(nil), nil
}

func (c *accrualCollector) collectCustomerFBOSelections(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
	featureKey string,
	target alpacadecimal.Decimal,
	asOf time.Time,
) ([]fboCollectionSelection, error) {
	sources, err := c.listCustomerFBOSources(ctx, customerID, currency, featureKey, asOf)
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(sources, cmpx.Compare[fboCollectionSource])

	return selectFBOSources(sources, target), nil
}

func (s fboCollectionSource) Compare(other fboCollectionSource) int {
	// TODO: Version this collection-order contract before changing it.
	// Existing ledger entries, corrections, and breakage releases assume this
	// priority/expiry/cursor ordering.
	if c := cmp.Compare(s.creditPriority, other.creditPriority); c != 0 {
		return c
	}

	if s.featureRestricted != other.featureRestricted {
		if s.featureRestricted {
			return -1
		}

		return 1
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
	featureKey string,
	asOf time.Time,
) ([]fboCollectionSource, error) {
	customerAccounts, err := c.deps.AccountService.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	if err := c.accountLocker.LockAccountsForPosting(ctx, []ledger.Account{customerAccounts.FBOAccount}); err != nil {
		return nil, fmt.Errorf("lock customer FBO account: %w", err)
	}

	fboAccountWithID, ok := customerAccounts.FBOAccount.(accountIdentifier)
	if !ok {
		return nil, fmt.Errorf("customer FBO account does not expose an ID")
	}

	sources, availableBySubAccountID, err := c.listCustomerFBOBalanceBucketSources(
		ctx,
		customerID.Namespace,
		fboAccountWithID.ID().ID,
		currency,
		featureKey,
		asOf,
	)
	if err != nil {
		return nil, err
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

	breakageSources := make([]fboCollectionSource, 0, len(openPlans)+len(sources))
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
		consumeFBOBalanceBucketSources(sources, plan.FBOSubAccountID, available)

		if !available.IsPositive() {
			continue
		}

		planCopy := plan
		expiresAt := plan.ExpiresAt
		route := plan.FBOAddress.Route().Route()
		if len(route.Features) > 0 && !lo.Contains(route.Features, featureKey) {
			continue
		}

		breakageSources = append(breakageSources, fboCollectionSource{
			address:           plan.FBOAddress,
			available:         available,
			creditPriority:    plan.CreditPriority,
			featureRestricted: len(route.Features) > 0,
			expiresAt:         &expiresAt,
			cursor:            plan.ID.ID,
			breakagePlan:      &planCopy,
		})
	}

	for _, source := range sources {
		if !source.available.IsPositive() {
			continue
		}

		breakageSources = append(breakageSources, source)
	}

	return breakageSources, nil
}

func (c *accrualCollector) listCustomerFBOBalanceBucketSources(
	ctx context.Context,
	namespace string,
	accountID string,
	currency currencyx.Code,
	featureKey string,
	asOf time.Time,
) ([]fboCollectionSource, map[string]alpacadecimal.Decimal, error) {
	route := ledger.RouteFilter{
		Currency: currency,
	}
	if featureKey != "" {
		route.MatchFeature = featureKey
	} else {
		route.Features = mo.Some([]string(nil))
	}

	buckets, err := c.deps.BalanceQuerier.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			AsOf:      &asOf,
			Route:     route,
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySourceChargeID},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get FBO balance buckets: %w", err)
	}

	sources := make([]fboCollectionSource, 0, len(buckets))
	availableBySubAccountID := make(map[string]alpacadecimal.Decimal, len(buckets))
	for _, bucket := range buckets {
		if !bucket.SettledAmount.IsPositive() {
			continue
		}

		route := bucket.Address.Route().Route()
		source := fboCollectionSource{
			address:           bucket.Address,
			sourceChargeID:    bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID],
			available:         bucket.SettledAmount,
			creditPriority:    customerFBOPriority(route),
			featureRestricted: len(route.Features) > 0,
			cursor:            fboBalanceBucketCursor(bucket),
		}
		availableBySubAccountID[bucket.Address.SubAccountID()] = availableBySubAccountID[bucket.Address.SubAccountID()].Add(bucket.SettledAmount)
		sources = append(sources, source)
	}

	return sources, availableBySubAccountID, nil
}

func fboBalanceBucketCursor(bucket ledger.BalanceBucket) string {
	sourceChargeID := "null"
	if value := bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID]; value != nil {
		sourceChargeID = *value
	}

	return bucket.Address.SubAccountID() + ":" + sourceChargeID
}

func consumeFBOBalanceBucketSources(sources []fboCollectionSource, subAccountID string, amount alpacadecimal.Decimal) {
	remaining := amount
	for i := range sources {
		if !remaining.IsPositive() {
			return
		}
		if sources[i].address.SubAccountID() != subAccountID || !sources[i].available.IsPositive() {
			continue
		}

		consumed := sources[i].available
		if consumed.GreaterThan(remaining) {
			consumed = remaining
		}
		sources[i].available = sources[i].available.Sub(consumed)
		remaining = remaining.Sub(consumed)
	}
}

func (s fboCollectionSelections) postingAmounts(spendChargeID *string) []transactions.PostingAmount {
	out := make([]transactions.PostingAmount, 0, len(s))

	for idx, selection := range s {
		collectionSource := strconv.Itoa(idx)
		out = append(out, transactions.PostingAmount{
			Address: selection.source.address,
			Amount:  selection.amount,
			Identity: ledger.EntryIdentityParts{
				CollectionSource: &collectionSource,
				SourceChargeID:   selection.source.sourceChargeID,
				SpendChargeID:    spendChargeID,
			},
			Annotations: models.Annotations{
				ledger.AnnotationCollectionSourceOrder: idx,
			},
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
	return fboCollectionSelections(selectFBOSources(sources, target)).postingAmounts(nil)
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

	return balance, nil
}
