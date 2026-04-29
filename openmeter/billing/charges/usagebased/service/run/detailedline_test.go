package run

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestMapRatingResultToRunDetailedLines(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	taxConfig := &productcatalog.TaxConfig{
		Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
	}

	in := usagebasedrating.GetDetailedRatingForUsageResult{
		DetailedLines: usagebased.DetailedLines{
			{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Namespace: "ns",
					Name:      "Usage",
				}),
				ServicePeriod:          servicePeriod,
				Currency:               currencyx.Code("USD"),
				ChildUniqueReferenceID: "unit-price-usage@[2025-01-01T00:00:00Z..2025-02-01T00:00:00Z]",
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				PerUnitAmount:          alpacadecimal.NewFromInt(3),
				Quantity:               alpacadecimal.NewFromInt(12),
				Category:               stddetailedline.CategoryRegular,
				TaxConfig:              taxConfig,
			},
		},
	}

	out := mapRatingResultToRunDetailedLines(in)
	require.Len(t, out, 1)
	require.Equal(t, "ns", out[0].Namespace)
	require.Equal(t, servicePeriod, out[0].ServicePeriod)
	require.Equal(t, currencyx.Code("USD"), out[0].Currency)
	require.Equal(t, productcatalog.InArrearsPaymentTerm, out[0].PaymentTerm)
	require.Equal(t, stddetailedline.CategoryRegular, out[0].Category)
	require.NotNil(t, out[0].TaxConfig)
	require.Equal(t, productcatalog.ExclusiveTaxBehavior, *out[0].TaxConfig.Behavior)
}
