package service

import (
	"context"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

func (s *service) getUsageBasedChargeQuantity(ctx context.Context, ch charges.Charge, featureMeters feature.FeatureMeters) (alpacadecimal.Decimal, error) {
	usageBasedIntent, err := ch.Intent.GetUsageBasedIntent()
	if err != nil {
		return alpacadecimal.Zero, err
	}

	featureMeter, err := featureMeters.Get(usageBasedIntent.FeatureKey, true)
	if err != nil {
		return alpacadecimal.Zero, err
	}

	cust, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: ch.Namespace,
			ID:        ch.Intent.CustomerID,
		},
	})
	if err != nil {
		return alpacadecimal.Zero, err
	}

	meterQueryParams := streaming.QueryParams{
		FilterCustomer: []streaming.Customer{cust},
		From:           &ch.Intent.ServicePeriod.From,
		To:             &ch.Intent.ServicePeriod.To,
		FilterGroupBy:  featureMeter.Feature.MeterGroupByFilters,
	}

	meterValues, err := s.streamingConnector.QueryMeter(
		ctx,
		ch.Namespace,
		*featureMeter.Meter,
		meterQueryParams)
	if err != nil {
		return alpacadecimal.Zero, err
	}

	return summarizeMeterQueryRow(meterValues), nil
}

func summarizeMeterQueryRow(in []meter.MeterQueryRow) alpacadecimal.Decimal {
	sum := alpacadecimal.Zero
	for _, row := range in {
		sum = sum.Add(alpacadecimal.NewFromFloat(row.Value))
	}

	return sum
}
