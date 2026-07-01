package collector

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/pkg/cmpx"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// collectCustomerFBOSelections builds the sources used by FBO->accrued
// collection:
// - query live FBO balance at route + source-charge granularity
// - if breakage is enabled, attach open breakage plans to matching source slices
// - select up to target from the prioritized sources
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

	// prioritize FBO sources before final collection.
	slices.SortStableFunc(sources, cmpx.Compare[fboCollectionSource])

	return selectFBOSources(sources, target), nil
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

	sources, err := c.listCustomerFBOBalanceBucketSources(
		ctx,
		customerID.Namespace,
		customerAccounts.FBOAccount.ID().ID,
		currency,
		featureKey,
		asOf,
	)
	if err != nil {
		return nil, err
	}

	// prioritize FBO sources before breakage reserves source balances.
	slices.SortStableFunc(sources, cmpx.Compare[fboCollectionSource])

	if c.breakage == nil {
		return sources, nil
	}

	return c.mapBreakagePlansToFBOCollectionSources(ctx, customerID, currency, featureKey, asOf, sources)
}

func (c *accrualCollector) mapBreakagePlansToFBOCollectionSources(
	ctx context.Context,
	customerID customer.CustomerID,
	currency currencyx.Code,
	featureKey string,
	asOf time.Time,
	sources []fboCollectionSource,
) ([]fboCollectionSource, error) {
	// Breakage plans decide which expiring credit is considered first. The FBO
	// balance buckets decide whether that plan still has live source balance
	// available to collect.
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
		reservedSources := reserveSourcesForBreakagePlan(sources, plan, featureKey)
		if len(reservedSources) == 0 {
			continue
		}

		planCopy := plan
		expiresAt := plan.ExpiresAt
		route := plan.FBOAddress.Route().Route()
		for _, reservedSource := range reservedSources {
			breakageSources = append(breakageSources, fboCollectionSource{
				address:           plan.FBOAddress,
				sourceChargeID:    reservedSource.sourceChargeID,
				available:         reservedSource.available,
				creditPriority:    plan.CreditPriority,
				freeCostBasis:     freeCostBasis(route.CostBasis),
				featureRestricted: len(route.Features) > 0,
				expiresAt:         &expiresAt,
				cursor:            plan.ID.ID + ":" + reservedSource.cursor,
				breakagePlan:      &planCopy,
			})
		}
	}

	for _, source := range sources {
		if !source.available.IsPositive() {
			continue
		}

		breakageSources = append(breakageSources, source)
	}

	return breakageSources, nil
}

// reserveSourcesForBreakagePlan assigns currently available FBO source balance
// to one open breakage plan in memory. The ledger write happens later, when the
// selected source is collected and its attached plan is released.
func reserveSourcesForBreakagePlan(
	sources []fboCollectionSource,
	plan breakage.Plan,
	featureKey string,
) []fboCollectionSource {
	route := plan.FBOAddress.Route().Route()
	if len(route.Features) > 0 && !lo.Contains(route.Features, featureKey) {
		return nil
	}

	if plan.SourceChargeID != nil {
		return reserveSourceIdentifiedBreakagePlan(sources, plan)
	}

	return reserveSourceUnknownBreakagePlan(sources, plan)
}

func (c *accrualCollector) listCustomerFBOBalanceBucketSources(
	ctx context.Context,
	namespace string,
	accountID string,
	currency currencyx.Code,
	featureKey string,
	asOf time.Time,
) ([]fboCollectionSource, error) {
	// Query at source-charge granularity. Route/sub-account balance alone is too
	// coarse once multiple purchased credit sources share the same FBO route.
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
		return nil, fmt.Errorf("get FBO balance buckets: %w", err)
	}

	sources := make([]fboCollectionSource, 0, len(buckets))
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
			freeCostBasis:     freeCostBasis(route.CostBasis),
			featureRestricted: len(route.Features) > 0,
			cursor:            fboBalanceBucketCursor(bucket),
		}
		sources = append(sources, source)
	}

	return sources, nil
}

func fboBalanceBucketCursor(bucket ledger.BalanceBucket) string {
	sourceChargeID := lo.FromPtrOr(bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID], "null")

	return bucket.Address.SubAccountID() + ":" + sourceChargeID
}

// reserveSourceIdentifiedBreakagePlan maps to at most one live source bucket
// because FBO balance buckets are grouped by sub-account + source_charge_id.
func reserveSourceIdentifiedBreakagePlan(sources []fboCollectionSource, plan breakage.Plan) []fboCollectionSource {
	if plan.SourceChargeID == nil {
		return nil
	}

	for i := range sources {
		if sources[i].address.SubAccountID() != plan.FBOSubAccountID {
			continue
		}
		if sources[i].sourceChargeID == nil || *sources[i].sourceChargeID != *plan.SourceChargeID {
			continue
		}

		reserved, ok := reserveFBOBalanceBucketSource(&sources[i], plan.OpenAmount)
		if !ok {
			return nil
		}

		return []fboCollectionSource{reserved}
	}

	return nil
}

// reserveSourceUnknownBreakagePlan handles source-less breakage records. Without
// source_charge_id, the plan may reserve from multiple live source buckets in
// its FBO sub-account, but the total reservation is capped by plan.OpenAmount.
func reserveSourceUnknownBreakagePlan(sources []fboCollectionSource, plan breakage.Plan) []fboCollectionSource {
	remaining := plan.OpenAmount
	reservedSources := make([]fboCollectionSource, 0)
	for i := range sources {
		if !remaining.IsPositive() {
			return reservedSources
		}
		if sources[i].address.SubAccountID() != plan.FBOSubAccountID {
			continue
		}

		reserved, ok := reserveFBOBalanceBucketSource(&sources[i], remaining)
		if !ok {
			continue
		}

		remaining = remaining.Sub(reserved.available)
		reservedSources = append(reservedSources, reserved)
	}

	return reservedSources
}

func reserveFBOBalanceBucketSource(source *fboCollectionSource, amount alpacadecimal.Decimal) (fboCollectionSource, bool) {
	if !source.available.IsPositive() || !amount.IsPositive() {
		return fboCollectionSource{}, false
	}

	reserved := source.available
	if reserved.GreaterThan(amount) {
		reserved = amount
	}
	source.available = source.available.Sub(reserved)

	out := *source
	out.available = reserved
	return out, true
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

func customerFBOPriority(route ledger.Route) int {
	if route.CreditPriority == nil {
		return ledger.DefaultCustomerFBOPriority
	}

	return *route.CreditPriority
}

func freeCostBasis(costBasis *alpacadecimal.Decimal) bool {
	return costBasis == nil || costBasis.IsZero()
}
