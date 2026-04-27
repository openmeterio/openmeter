package run

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
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

	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: chargesmeta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "ns"},
				ID:              "charge-1",
			},
			Intent: usagebased.Intent{
				Intent: chargesmeta.Intent{
					Currency:      currencyx.Code("USD"),
					ServicePeriod: servicePeriod,
					TaxConfig: &productcatalog.TaxConfig{
						Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					},
				},
			},
		},
	}

	run := usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
				Namespace: "ns",
				ID:        "run-1",
			},
		},
	}

	in := usagebasedrating.GetDetailedRatingForUsageResult{
		GenerateDetailedLinesResult: billingrating.GenerateDetailedLinesResult{
			DetailedLines: billingrating.DetailedLines{
				{
					Name:                   "Usage",
					Quantity:               alpacadecimal.NewFromInt(12),
					PerUnitAmount:          alpacadecimal.NewFromInt(3),
					ChildUniqueReferenceID: "unit-price-usage@[2025-01-01T00:00:00Z..2025-02-01T00:00:00Z]",
				},
			},
		},
	}

	out := mapRatingResultToRunDetailedLines(charge, run, in)
	require.Len(t, out, 1)
	require.Equal(t, "ns", out[0].Namespace)
	require.Equal(t, servicePeriod, out[0].ServicePeriod)
	require.Equal(t, currencyx.Code("USD"), out[0].Currency)
	require.Equal(t, productcatalog.InArrearsPaymentTerm, out[0].PaymentTerm)
	require.Equal(t, stddetailedline.CategoryRegular, out[0].Category)
	require.NotNil(t, out[0].TaxConfig)
	require.NotSame(t, charge.Intent.TaxConfig, out[0].TaxConfig)
}
