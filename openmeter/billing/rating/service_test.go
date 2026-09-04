package rating_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type gatheringLine struct {
	price *productcatalog.Price
}

func (l gatheringLine) GetPrice() *productcatalog.Price {
	return l.price
}

func (gatheringLine) GetServicePeriod() timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{}
}

func (gatheringLine) GetSplitLineGroupID() *string {
	return nil
}

func (gatheringLine) GetInvoiceAt() time.Time {
	return time.Time{}
}

func (gatheringLine) GetID() string {
	return "line"
}

func TestResolveBillablePeriodInputValidate(t *testing.T) {
	flatLine := gatheringLine{
		price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount: alpacadecimal.NewFromInt(1),
		}),
	}
	meteredLine := gatheringLine{
		price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(1),
		}),
	}
	asOf := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)

	t.Run("flat lines do not require feature dependencies", func(t *testing.T) {
		err := (rating.ResolveBillablePeriodInput{
			AsOf: asOf,
			Line: flatLine,
		}).Validate()

		require.NoError(t, err)
	})

	t.Run("metered lines require a feature and meter", func(t *testing.T) {
		err := (rating.ResolveBillablePeriodInput{
			AsOf: asOf,
			Line: meteredLine,
		}).Validate()

		require.ErrorContains(t, err, "feature is required for metered lines")
		require.ErrorContains(t, err, "meter is required for metered lines")
	})

	t.Run("rating does not validate the feature meter association", func(t *testing.T) {
		differentMeterID := "different-meter"
		err := (rating.ResolveBillablePeriodInput{
			AsOf: asOf,
			Line: meteredLine,
			Feature: &feature.Feature{
				MeterID: &differentMeterID,
			},
			Meter: &meter.Meter{},
		}).Validate()

		require.NoError(t, err)
	})
}
