package customerbalance

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// featureBucketKey identifies one exact feature-restriction set in a balance
// breakdown. The empty key is the unrestricted bucket. Keys are built from the
// sorted, deduplicated feature list so equal restriction sets always collapse
// into the same bucket.
type featureBucketKey string

const featureBucketKeySeparator = "\x1f"

func newFeatureBucketKey(features []string) featureBucketKey {
	if len(features) == 0 {
		return ""
	}

	return featureBucketKey(strings.Join(slicesx.Normalize(features), featureBucketKeySeparator))
}

func (k featureBucketKey) Features() []string {
	if k == "" {
		return nil
	}

	return strings.Split(string(k), featureBucketKeySeparator)
}

// FeatureBucketBalance is one slice of a customer's balance scoped to an exact
// feature-restriction set. Features is empty for the unrestricted bucket.
type FeatureBucketBalance struct {
	Features []string
	Balance  Balance
}

// GetBalanceWithFeatureBreakdown computes the balance partitioned by
// feature-restriction set alongside the aggregate. The aggregate is derived by
// summing the buckets, so bucket amounts reconcile to the totals by
// construction; both run off a single enumeration of credit sources, live
// charge impacts, and pending grants rather than one recomputation per bucket.
func (s *service) GetBalanceWithFeatureBreakdown(ctx context.Context, input GetBalanceServiceInput) (Balance, []FeatureBucketBalance, error) {
	if err := input.Validate(); err != nil {
		return nil, nil, err
	}

	settledByBucket, err := s.getSettledBalanceBuckets(ctx, input)
	if err != nil {
		return nil, nil, err
	}

	sources, err := s.getLiveBalanceSources(ctx, input)
	if err != nil {
		return nil, nil, err
	}

	impacts, err := s.getChargeLiveBalanceImpacts(ctx, input.CustomerID, input.Currency, normalizeFeatureFilter(input.FeatureFilter))
	if err != nil {
		return nil, nil, fmt.Errorf("get charge live balance impacts: %w", err)
	}

	liveImpactsByBucket := s.balanceCalculator.CalculateLiveImpactBuckets(sources, impacts)

	pendingByBucket, err := s.getPendingGrantAmountBuckets(ctx, input.CustomerID, input.Currency, normalizeFeatureFilter(input.FeatureFilter), input.pendingGrantAsOf())
	if err != nil {
		return nil, nil, fmt.Errorf("get pending grant amount: %w", err)
	}

	keys := make(map[featureBucketKey]struct{})
	for key := range settledByBucket {
		keys[key] = struct{}{}
	}
	for key := range liveImpactsByBucket {
		keys[key] = struct{}{}
	}
	for key := range pendingByBucket {
		keys[key] = struct{}{}
	}

	// Unrestricted bucket first, then lexicographic by restriction set.
	orderedKeys := make([]featureBucketKey, 0, len(keys))
	for key := range keys {
		orderedKeys = append(orderedKeys, key)
	}
	slices.SortFunc(orderedKeys, func(a, b featureBucketKey) int {
		return cmp.Compare(a, b)
	})

	total := balance{
		settled: alpacadecimal.Zero,
		live:    alpacadecimal.Zero,
		pending: alpacadecimal.Zero,
	}
	buckets := make([]FeatureBucketBalance, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		bucket := balance{
			settled: settledByBucket[key],
			live:    settledByBucket[key].Sub(liveImpactsByBucket[key]),
			pending: pendingByBucket[key],
		}

		total.settled = total.settled.Add(bucket.settled)
		total.live = total.live.Add(bucket.live)
		total.pending = total.pending.Add(bucket.pending)

		buckets = append(buckets, FeatureBucketBalance{
			Features: key.Features(),
			Balance:  bucket,
		})
	}

	return total, buckets, nil
}

// getSettledBalanceBuckets itemizes the settled balance (booked FBO credit
// plus open receivable advances) per sub-account feature-restriction set,
// honoring the same route filters as getSettledBalance.
func (s *service) getSettledBalanceBuckets(ctx context.Context, input GetBalanceServiceInput) (map[featureBucketKey]alpacadecimal.Decimal, error) {
	query := input.balanceQuery()

	customerAccounts, err := s.AccountResolver.GetCustomerAccounts(ctx, input.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("get customer accounts: %w", err)
	}

	multiKeys, hasMultiKeys := multiFeatureKeys(normalizeFeatureFilter(input.FeatureFilter))

	buckets := make(map[featureBucketKey]alpacadecimal.Decimal)
	for _, scope := range []struct {
		account ledger.Account
		filter  ledger.RouteFilter
	}{
		{account: customerAccounts.FBOAccount, filter: input.bookedRoute()},
		{account: customerAccounts.ReceivableAccount, filter: input.advanceRoute()},
	} {
		subAccounts, err := s.SubAccountService.ListSubAccounts(ctx, ledger.ListSubAccountsInput{
			Namespace: scope.account.ID().Namespace,
			AccountID: scope.account.ID().ID,
		})
		if err != nil {
			return nil, fmt.Errorf("list sub accounts: %w", err)
		}

		for _, subAccount := range subAccounts {
			route := subAccount.Route()
			if !route.Matches(scope.filter) {
				continue
			}

			if hasMultiKeys && !routeMatchesAnyFeature(route, multiKeys) {
				continue
			}

			subAccountBalance, err := s.BalanceQuerier.GetSubAccountBalance(ctx, subAccount, query)
			if err != nil {
				return nil, fmt.Errorf("get sub account balance: %w", err)
			}

			if subAccountBalance.IsZero() {
				continue
			}

			key := newFeatureBucketKey(route.Features)
			buckets[key] = buckets[key].Add(subAccountBalance)
		}
	}

	return buckets, nil
}
