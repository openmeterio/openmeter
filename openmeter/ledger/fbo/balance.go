package fbo

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

// SpendableBalanceBucketQuery selects complete FBO sources. GroupBy may contain
// source_charge_id; omitting it sums independently spendable sources per subaccount.
// AsOf is the time at which credit must be available.
type SpendableBalanceBucketQuery ledger.BalanceBucketQuery

var _ models.Validator = SpendableBalanceBucketQuery{}

func (q SpendableBalanceBucketQuery) Validate() error {
	var errs []error
	if err := (ledger.MinimumBalanceBucketQuery{BalanceBucketQuery: ledger.BalanceBucketQuery(q)}).Validate(); err != nil {
		errs = append(errs, err)
	}

	if q.Filters.AccountID == nil || *q.Filters.AccountID == "" {
		errs = append(errs, errors.New("FBO account id is required"))
	}

	if q.Filters.TransactionID != nil || q.Filters.Provenance.SpendChargeID.IsPresent() || q.Filters.Provenance.CollectionOriginID.IsPresent() || len(q.ExcludeAnnotationFilters) > 0 {
		errs = append(errs, errors.New("spendable balance requires complete source histories"))
	}

	for _, dimension := range q.GroupBy {
		if dimension != ledger.BalanceBucketGroupBySourceChargeID {
			errs = append(errs, fmt.Errorf("unsupported spendable balance grouping: %s", dimension))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// GetSpendableBalanceBuckets preserves recorded future spending. Only future
// breakage is excluded: consuming credit releases that scheduled breakage in the
// same group. Breakage at or before AsOf still reduces the opening balance.
func (s *service) GetSpendableBalanceBuckets(ctx context.Context, query SpendableBalanceBucketQuery) ([]ledger.BalanceBucket, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	account, err := s.dependencies.AccountCatalog.GetAccountByID(ctx, models.NamespacedID{
		Namespace: query.Namespace,
		ID:        *query.Filters.AccountID,
	})
	if err != nil {
		return nil, err
	}

	if account.Type() != ledger.AccountTypeCustomerFBO {
		return nil, models.NewGenericValidationError(errors.New("spendable balance requires a customer FBO account"))
	}

	perSource := ledger.BalanceBucketQuery(query)
	perSource.GroupBy = []string{ledger.BalanceBucketGroupBySourceChargeID}
	buckets, err := s.dependencies.BalanceQuerier.GetMinimumBalanceBuckets(ctx, ledger.MinimumBalanceBucketQuery{
		BalanceBucketQuery: perSource,
		ExcludeFutureAnnotationFilters: map[string]string{
			ledger.AnnotationCollectionType: ledger.CollectionTypeBreakage,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query spendable FBO balance: %w", err)
	}

	// Collection cannot borrow capacity between sources. Clamp each source before
	// aggregating, rather than taking the minimum of their combined journal.
	spendable := make([]ledger.BalanceBucket, 0, len(buckets))
	bySubaccount := make(map[string]int)
	for _, bucket := range buckets {
		if bucket.SettledAmount.IsNegative() {
			s.reportNegativeSourceBalance(ctx, query, bucket)
		}

		if !bucket.SettledAmount.IsPositive() {
			continue
		}

		if slices.Contains(query.GroupBy, ledger.BalanceBucketGroupBySourceChargeID) {
			spendable = append(spendable, bucket)
			continue
		}

		id := bucket.Address.SubAccountID()
		if idx, ok := bySubaccount[id]; ok {
			combined := &spendable[idx]
			combined.SettledAmount = combined.SettledAmount.Add(bucket.SettledAmount)
			combined.PendingAmount = combined.PendingAmount.Add(bucket.PendingAmount)
			if bucket.OldestMatchingEntryCreatedAt.Before(combined.OldestMatchingEntryCreatedAt) {
				combined.OldestMatchingEntryCreatedAt = bucket.OldestMatchingEntryCreatedAt
			}
		} else {
			bucket.GroupByValues = map[string]*string{}
			bySubaccount[id] = len(spendable)
			spendable = append(spendable, bucket)
		}
	}

	return spendable, nil
}

// Repeated reads count again: these are observations, not distinct broken sources.
func (s *service) reportNegativeSourceBalance(ctx context.Context, query SpendableBalanceBucketQuery, bucket ledger.BalanceBucket) {
	s.negativeSourceObservations.Add(ctx, 1)

	s.logger.WarnContext(ctx, "negative FBO source minimum balance observed",
		"namespace", query.Namespace,
		"account_id", *query.Filters.AccountID,
		"sub_account_id", bucket.Address.SubAccountID(),
		"source_charge_id", bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID],
		"as_of", *query.Filters.AsOf,
		"minimum_balance", bucket.SettledAmount.String(),
	)
}
