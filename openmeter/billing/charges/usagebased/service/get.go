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
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/ref"
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

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]usagebased.Charge, error) {
		charges, err := s.adapter.GetByIDs(ctx, input)
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
	})
}

func (s *service) GetByID(ctx context.Context, input usagebased.GetByIDInput) (usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return usagebased.Charge{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (usagebased.Charge, error) {
		charge, err := s.adapter.GetByID(ctx, input)
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
		}

		return charge, nil
	})
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
	referencedFeatureMeters := lo.Uniq(lo.Map(charges, func(charge usagebased.Charge, _ int) ref.IDOrKey {
		return charge.GetFeatureKeyOrID()
	}))

	featureMeters, err := s.featureService.ResolveFeatureMeters(ctx, namespace, referencedFeatureMeters...)
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

			var dueTotals totals.Totals
			dueTotals, err = s.rater.GetTotalsForUsage(ctx, usagebasedrating.GetRatingForUsageInput{
				Charge:         charge,
				Customer:       customerOverridesById[charge.GetCustomerID()],
				FeatureMeter:   featureMeter,
				StoredAtOffset: storedAt,
			})
			if err != nil {
				err = fmt.Errorf("get totals for charge %s: %w", charge.ID, err)
				return
			}

			ratingResults.Store(charge.GetChargeID(), dueTotals)
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
		dueTotalsAny, ok := ratingResults.Load(charge.GetChargeID())
		if !ok {
			return charge, fmt.Errorf("totals result not found for charge %s", charge.ID)
		}

		dueTotals, ok := dueTotalsAny.(totals.Totals)
		if !ok {
			return charge, fmt.Errorf("invalid totals type for charge %s", charge.ID)
		}

		charge.Expands.RealtimeUsage = &dueTotals
		return charge, nil
	})
}
