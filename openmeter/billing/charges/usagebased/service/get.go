package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/samber/lo"
	"golang.org/x/sync/semaphore"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

const (
	// defaultMaxParallelRatingsPerRequest is the number of workers to use for the rating (fetching from CH).
	defaultMaxParallelRatingsPerRequest = 5
)

func (s *service) GetByIDs(ctx context.Context, input usagebased.GetByIDsInput) ([]usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// The realtime usage expansion queries ClickHouse per charge, so it runs
	// after the adapter read instead of inside the transaction: a remote call
	// must not pin the pooled Postgres connection. Callers that wrap this in
	// their own transaction hoist the expansion the same way.
	charges, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]usagebased.Charge, error) {
		return s.adapter.GetByIDs(ctx, input)
	})
	if err != nil {
		return nil, err
	}

	if input.Expands.Has(meta.ExpandRealtimeUsage) {
		charges, err = s.expandChargesUsage(ctx, input.Namespace, charges)
		if err != nil {
			return nil, err
		}
	}

	return charges, nil
}

func (s *service) GetByID(ctx context.Context, input usagebased.GetByIDInput) (usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return usagebased.Charge{}, err
	}

	// The realtime totals read rates against ClickHouse, so it runs after
	// the adapter read instead of inside the transaction; see GetByIDs.
	charge, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) (usagebased.Charge, error) {
		return s.adapter.GetByID(ctx, input)
	})
	if err != nil {
		return usagebased.Charge{}, err
	}

	if input.Expands.Has(meta.ExpandRealtimeUsage) {
		expanded, err := s.expandChargesUsage(ctx, charge.Namespace, usagebased.Charges{charge})
		if err != nil {
			return usagebased.Charge{}, err
		}

		charge = expanded[0]
	}

	return charge, nil
}

func (s *service) expandChargesUsage(ctx context.Context, namespace string, charges usagebased.Charges) (usagebased.Charges, error) {
	storedAt := clock.Now()
	var eligible usagebased.Charges
	var eligibleIndexes []int
	for idx, charge := range charges {
		if charge.GetIntentDeletedAt() != nil {
			continue
		}

		status, err := charge.Status.ToMetaChargeStatus()
		if err != nil {
			return nil, fmt.Errorf("get status for charge %s: %w", charge.ID, err)
		}

		if status == meta.ChargeStatusActive ||
			(status == meta.ChargeStatusCreated && charge.Intent.GetEffectiveServicePeriod().Contains(storedAt)) {
			eligible = append(eligible, charge)
			eligibleIndexes = append(eligibleIndexes, idx)
		}
	}
	if len(eligible) == 0 {
		return charges, nil
	}

	uniqueCustomers := lo.Uniq(lo.Map(eligible, func(charge usagebased.Charge, _ int) customer.CustomerID {
		return charge.GetCustomerID()
	}))

	customerOverridesById := make(map[customer.CustomerID]billing.CustomerBillingProfile)
	for _, customerID := range uniqueCustomers {
		customerOverride, err := s.customerOverrideService.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
			Customer: customerID,
			Expand: billing.CustomerBillingProfileExpand{
				Customer: true,
			},
		})
		if err != nil {
			return nil, err
		}

		customerOverridesById[customerID] = customerOverride
	}

	// Fetch all referenced feature meters in bulk.
	featureMeters, err := s.featureMeterResolver.Resolve(ctx, namespace, eligible...)
	if err != nil {
		return nil, err
	}

	sem := semaphore.NewWeighted(int64(defaultMaxParallelRatingsPerRequest))
	errCh := make(chan error, len(eligible))
	expanded := slices.Clone(charges)

	var wg sync.WaitGroup

	for idx, charge := range eligible {
		featureMeter, err := featureMeters.Get(charge)
		if err != nil {
			errCh <- fmt.Errorf("resolving feature meter: %w", err)
			break
		}

		err = sem.Acquire(ctx, 1)
		if err != nil {
			// Clean up and stop the loop
			errCh <- fmt.Errorf("acquiring worker slot: %w", err)
			break
		}

		wg.Go(func() {
			defer sem.Release(1)
			var err error
			defer func() {
				if err != nil {
					errCh <- err
				}
			}()

			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("rating charge %s: %v", charge.ID, r)
				}
			}()

			ratedUsage, err := s.rater.GetTotalsForUsage(ctx, usagebasedrating.GetTotalsForUsageInput{
				Charge:                  charge,
				Customer:                customerOverridesById[charge.GetCustomerID()],
				FeatureMeter:            featureMeter,
				StoredAtLT:              storedAt,
				IgnoreMinimumCommitment: storedAt.Before(charge.Intent.GetEffectiveServicePeriod().To),
			})
			issues, err := billing.ToValidationIssues(err, billing.RequireWarningsOnly())
			if err != nil {
				err = fmt.Errorf("get totals for charge %s: %w", charge.ID, err)
				return
			}

			charge.Expands.RealtimeUsage = &ratedUsage.Totals
			charge.Expands.RealtimeQuantity = &ratedUsage.MeteredQuantity
			expanded[eligibleIndexes[idx]] = withRealtimeValidationIssues(charge, issues)
		})
	}

	wg.Wait()

	close(errCh)

	var errs []error

	for err := range errCh {
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return expanded, nil
}

// Realtime warnings describe this read's metered snapshot, not persisted charge state.
func withRealtimeValidationIssues(charge usagebased.Charge, issues billing.ValidationIssues) usagebased.Charge {
	charge.ValidationIssues, _ = replaceValidationIssueComponent(
		charge.ValidationIssues,
		billing.ValidationComponentBillingRating,
		issues,
	)

	return charge
}
