package fbo

import (
	"context"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/pkg/cmpx"
)

func (c *service) listCustomerFBOSources(
	ctx context.Context,
	query SourceQuery,
	scope *Scope,
) ([]source, error) {
	customerAccounts, err := c.dependencies.AccountService.GetCustomerAccounts(ctx, query.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	if err := customerAccounts.LockForPosting(ctx, c.accountLocker); err != nil {
		return nil, fmt.Errorf("lock customer FBO account: %w", err)
	}

	sources, err := c.listCustomerFBOBalanceBucketSources(
		ctx,
		query,
		customerAccounts.FBOAccount.ID().ID,
	)
	if err != nil {
		return nil, err
	}

	if scope != nil {
		for idx := range sources {
			key := keyForSource(sources[idx].address, sources[idx].sourceChargeID)
			sources[idx].available = sources[idx].available.Sub(scope.reserved[key])
		}
	}

	// prioritize FBO sources before breakage reserves source balances.
	slices.SortStableFunc(sources, cmpx.Compare[source])

	sources, err = c.mapBreakagePlansToFBOCollectionSources(ctx, query, sources)
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(sources, cmpx.Compare[source])

	return sources, nil
}

func (c *service) mapBreakagePlansToFBOCollectionSources(
	ctx context.Context,
	query SourceQuery,
	sources []source,
) ([]source, error) {
	// Breakage plans decide which expiring credit is considered first. The FBO
	// balance buckets decide whether that plan still has live source balance
	// available to collect.
	openPlans, err := c.breakage.ListPlans(ctx, breakage.ListPlansInput{
		CustomerID: query.CustomerID,
		Currency:   query.Currency.Code,
		AsOf:       query.AsOf,
	})
	if err != nil {
		return nil, fmt.Errorf("list open breakage plans: %w", err)
	}

	breakageSources := make([]source, 0, len(openPlans)+len(sources))
	for _, plan := range openPlans {
		// ListPlans filters by code; keep the exact managed identity
		// check here until same-code plan locking warrants an adapter-level filter.
		reservedSources := reserveSourcesForBreakagePlan(sources, plan, query.Currency, ledger.Route{Filters: query.Filters})
		if len(reservedSources) == 0 {
			continue
		}

		planCopy := plan
		expiresAt := plan.ExpiresAt
		route := plan.FBOAddress.Route().Route()
		for _, reservedSource := range reservedSources {
			breakageSources = append(breakageSources, source{
				address:        plan.FBOAddress,
				sourceChargeID: reservedSource.sourceChargeID,
				available:      reservedSource.available,
				creditPriority: plan.CreditPriority,
				restricted:     !route.Filters.IsEmpty(),
				expiresAt:      &expiresAt,
				cursor:         plan.ID.ID + ":" + reservedSource.cursor,
				breakagePlan:   &planCopy,
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
	sources []source,
	plan breakage.Plan,
	currency currencies.CurrencyReference,
	targetRoute ledger.Route,
) []source {
	route := plan.FBOAddress.Route().Route()
	if !route.Currency.Equal(currency) {
		return nil
	}

	if !route.Filters.Matches(targetRoute) {
		return nil
	}

	if plan.SourceChargeID != nil {
		return reserveSourceIdentifiedBreakagePlan(sources, plan)
	}

	return reserveSourceUnknownBreakagePlan(sources, plan)
}

func (c *service) listCustomerFBOBalanceBucketSources(
	ctx context.Context,
	query SourceQuery,
	accountID string,
) ([]source, error) {
	// Query at source-charge granularity. Route/sub-account balance alone is too
	// coarse once multiple purchased credit sources share the same FBO route.
	route := ledger.RouteFilter{
		Currency: query.Currency,
	}
	targetRoute := ledger.Route{Filters: query.Filters}
	if len(targetRoute.Filters.Features) == 1 {
		route.MatchFeature = targetRoute.Filters.Features[0]
	} else if len(targetRoute.Filters.Features) == 0 {
		route.Features = mo.Some([]string(nil))
	}

	buckets, err := c.dependencies.BalanceQuerier.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: query.CustomerID.Namespace,
		Filters: ledger.Filters{
			AccountID: &accountID,
			AsOf:      &query.AsOf,
			Route:     route,
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySourceChargeID},
	})
	if err != nil {
		return nil, fmt.Errorf("get FBO balance buckets: %w", err)
	}

	sources := make([]source, 0, len(buckets))
	for _, bucket := range buckets {
		if !bucket.SettledAmount.IsPositive() {
			continue
		}

		route := bucket.Address.Route().Route()
		if !route.Filters.Matches(targetRoute) {
			continue
		}

		source := source{
			address:        bucket.Address,
			sourceChargeID: bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID],
			available:      bucket.SettledAmount,
			creditPriority: customerFBOPriority(route),
			restricted:     !route.Filters.IsEmpty(),
			cursor:         balanceBucketCursor(bucket),
		}
		sources = append(sources, source)
	}

	return sources, nil
}

func balanceBucketCursor(bucket ledger.BalanceBucket) string {
	sourceChargeID := lo.FromPtrOr(bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID], "null")

	return bucket.Address.SubAccountID() + ":" + sourceChargeID
}

// reserveSourceIdentifiedBreakagePlan maps to at most one live source bucket
// because FBO balance buckets are grouped by sub-account + source_charge_id.
func reserveSourceIdentifiedBreakagePlan(sources []source, plan breakage.Plan) []source {
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

		return []source{reserved}
	}

	return nil
}

// reserveSourceUnknownBreakagePlan handles source-less breakage records. Without
// source_charge_id, the plan may reserve from multiple live source buckets in
// its FBO sub-account, but the total reservation is capped by plan.OpenAmount.
func reserveSourceUnknownBreakagePlan(sources []source, plan breakage.Plan) []source {
	remaining := plan.OpenAmount
	reservedSources := make([]source, 0)
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

func reserveFBOBalanceBucketSource(bucketSource *source, amount alpacadecimal.Decimal) (source, bool) {
	if !bucketSource.available.IsPositive() || !amount.IsPositive() {
		return source{}, false
	}

	reserved := bucketSource.available
	if reserved.GreaterThan(amount) {
		reserved = amount
	}

	bucketSource.available = bucketSource.available.Sub(reserved)

	out := *bucketSource
	out.available = reserved

	return out, true
}

func selectFBOSources(sources []source, target alpacadecimal.Decimal) []selection {
	remaining := target
	out := make([]selection, 0, len(sources))

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

		out = append(out, selection{
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
