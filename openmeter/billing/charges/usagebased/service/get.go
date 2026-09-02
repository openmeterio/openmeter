package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/samber/lo"
	"golang.org/x/sync/semaphore"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/slicesx"
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
		totals, err := s.GetCurrentTotals(ctx, usagebased.GetCurrentTotalsInput{
			ChargeID: charge.GetChargeID(),
		})
		if err != nil {
			return usagebased.Charge{}, err
		}

		charge.Expands.RealtimeUsage = &totals.DueTotals
		charge.Expands.RealtimeQuantity = &totals.MeteredQuantity
	}

	return charge, nil
}

func (s *service) expandChargesUsage(ctx context.Context, namespace string, charges usagebased.Charges) (usagebased.Charges, error) {
	// Fetch unique customers from the charges to avoid duplicate calls to the customer override service.
	uniqueCustomers := lo.Uniq(lo.Map(charges, func(charge usagebased.Charge, _ int) customer.CustomerID {
		return charge.GetCustomerID()
	}))

	customerOverridesById := make(map[customer.CustomerID]billing.CustomerOverrideWithDetails)
	for _, customerID := range uniqueCustomers {
		customerOverride, err := s.customerOverrideService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customerID,
			Expand: billing.CustomerOverrideExpand{
				Customer: true,
			},
		})
		if err != nil {
			return nil, err
		}
		customerOverridesById[customerID] = customerOverride
	}

	// Fetch all references featureMeters in bulk
	referencedFeatureMeters := lo.Uniq(lo.Map(charges, func(charge usagebased.Charge, _ int) billingfeaturemeter.FeatureMeterRef {
		return billingfeaturemeter.FeatureMeterRef{
			IDOrKey:      charge.GetFeatureKeyOrID(),
			RequireMeter: true,
		}
	}))

	featureMeters, err := s.featureMeterResolver.Resolve(ctx, namespace, referencedFeatureMeters)
	if err != nil {
		return nil, err
	}

	// Let's do the rating for each charge
	sem := semaphore.NewWeighted(int64(defaultMaxParallelRatingsPerRequest))
	storedAt := clock.Now()

	errCh := make(chan error, len(charges))
	ratingResults := sync.Map{}

	var wg sync.WaitGroup

	for _, charge := range charges {
		featureMeter, err := charge.ResolveFeatureMeter(featureMeters)
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

			var ratedUsage usagebasedrating.GetTotalsForUsageResult
			ratedUsage, err = s.rater.GetTotalsForUsage(ctx, usagebasedrating.GetTotalsForUsageInput{
				Charge:                  charge,
				Customer:                customerOverridesById[charge.GetCustomerID()],
				FeatureMeter:            featureMeter,
				StoredAtLT:              storedAt,
				IgnoreMinimumCommitment: storedAt.Before(charge.Intent.GetEffectiveServicePeriod().To),
			})
			if err != nil {
				err = fmt.Errorf("get totals for charge %s: %w", charge.ID, err)
				return
			}

			ratingResults.Store(charge.GetChargeID(), ratedUsage)
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

	return slicesx.MapWithErr(charges, func(charge usagebased.Charge) (usagebased.Charge, error) {
		ratedAny, ok := ratingResults.Load(charge.GetChargeID())
		if !ok {
			return charge, fmt.Errorf("totals result not found for charge %s", charge.ID)
		}

		rated, ok := ratedAny.(usagebasedrating.GetTotalsForUsageResult)
		if !ok {
			return charge, fmt.Errorf("invalid totals type for charge %s", charge.ID)
		}

		charge.Expands.RealtimeUsage = &rated.Totals
		charge.Expands.RealtimeQuantity = &rated.MeteredQuantity
		return charge, nil
	})
}
