package usagebasedrating

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type SnapshotQuantityInput struct {
	Customer       streaming.Customer
	FeatureMeter   feature.FeatureMeter
	ServicePeriod  timeutil.ClosedPeriod
	StoredAtOffset time.Time
}

func (i SnapshotQuantityInput) Validate() error {
	if i.FeatureMeter.Meter == nil {
		return fmt.Errorf("meter is required")
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		return fmt.Errorf("service period: %w", err)
	}

	if i.StoredAtOffset.IsZero() {
		return fmt.Errorf("stored at offset is required")
	}

	return nil
}

func (s *service) snapshotQuantity(ctx context.Context, in SnapshotQuantityInput) (alpacadecimal.Decimal, error) {
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
				Lt: &in.StoredAtOffset,
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
