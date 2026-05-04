package rating

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type getQuantityForUsageInput struct {
	Charge          usagebased.Charge
	Customer        billing.CustomerOverrideWithDetails
	FeatureMeter    feature.FeatureMeter
	ServicePeriodTo time.Time
	StoredAtLT      time.Time
}

func (i getQuantityForUsageInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if i.Customer.Customer == nil {
		return fmt.Errorf("customer is required")
	}

	if i.FeatureMeter.Meter == nil {
		return fmt.Errorf("feature meter is required")
	}

	if i.ServicePeriodTo.IsZero() {
		return fmt.Errorf("service period to is required")
	}

	period := i.Charge.Intent.ServicePeriod
	if !i.ServicePeriodTo.After(period.From) {
		return fmt.Errorf("service period to must be after charge service period from")
	}

	if i.ServicePeriodTo.After(period.To) {
		return fmt.Errorf("service period to must not be after charge service period to")
	}

	if i.StoredAtLT.IsZero() {
		return fmt.Errorf("stored at lt is required")
	}

	return nil
}

func (s *service) getQuantityForUsage(ctx context.Context, in getQuantityForUsageInput) (alpacadecimal.Decimal, error) {
	if err := in.Validate(); err != nil {
		return alpacadecimal.Zero, err
	}

	servicePeriod := in.Charge.Intent.ServicePeriod
	servicePeriod.To = in.ServicePeriodTo

	return s.snapshotQuantity(ctx, snapshotQuantityInput{
		Customer:      in.Customer.Customer,
		FeatureMeter:  in.FeatureMeter,
		ServicePeriod: servicePeriod,
		StoredAtLT:    in.StoredAtLT,
	})
}

type snapshotQuantityInput struct {
	Customer      streaming.Customer
	FeatureMeter  feature.FeatureMeter
	ServicePeriod timeutil.ClosedPeriod
	StoredAtLT    time.Time
}

func (i snapshotQuantityInput) Validate() error {
	if i.FeatureMeter.Meter == nil {
		return fmt.Errorf("meter is required")
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		return fmt.Errorf("service period: %w", err)
	}

	if i.StoredAtLT.IsZero() {
		return fmt.Errorf("stored at lt is required")
	}

	return nil
}

func (s *service) snapshotQuantity(ctx context.Context, in snapshotQuantityInput) (alpacadecimal.Decimal, error) {
	if err := in.Validate(); err != nil {
		return alpacadecimal.Zero, billing.ValidationError{
			Err: err,
		}
	}

	meterQueryParams := streaming.QueryParams{
		FilterCustomer: []streaming.Customer{in.Customer},
		From:           &in.ServicePeriod.From,
		To:             &in.ServicePeriod.To,
		FilterGroupBy:  in.FeatureMeter.Feature.MeterGroupByFilters,
		FilterStoredAt: &filter.FilterTimeUnix{
			FilterTime: filter.FilterTime{
				Lt: &in.StoredAtLT,
			},
		},
	}

	res, err := s.streamingConnector.QueryMeter(ctx, in.FeatureMeter.Feature.Namespace, *in.FeatureMeter.Meter, meterQueryParams)
	if err != nil {
		return alpacadecimal.Zero, fmt.Errorf("querying meter: %w", err)
	}

	return summarizeMeterQueryRow(res), nil
}

func summarizeMeterQueryRow(in []meter.MeterQueryRow) alpacadecimal.Decimal {
	sum := alpacadecimal.Zero
	for _, row := range in {
		sum = sum.Add(alpacadecimal.NewFromFloat(row.Value))
	}

	return sum
}
