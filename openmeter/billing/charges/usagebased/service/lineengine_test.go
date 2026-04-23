package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestLineEngineIsLineBillableAsOfSupportsPartialBillablePeriod(t *testing.T) {
	engine := &LineEngine{}

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	billablePeriod := timeutil.ClosedPeriod{
		From: servicePeriod.From,
		To:   time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
	}

	billable, err := engine.IsLineBillableAsOf(t.Context(), billing.IsLineBillableAsOfInput{
		AsOf:                   billablePeriod.To,
		ProgressiveBilling:     true,
		ResolvedBillablePeriod: billablePeriod,
	})
	require.NoError(t, err)
	require.True(t, billable)
}

func TestLineEngineSplitGatheringLineKeepsChargeGroupingWithoutChildReferences(t *testing.T) {
	engine := &LineEngine{}

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	splitAt := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)

	price := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(2),
	})

	line := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "test",
				ID:        "01K00000000000000000000001",
				Name:      "usage",
			}),
			ManagedBy:              billing.SystemManagedLine,
			Engine:                 billing.LineEngineTypeChargeUsageBased,
			Currency:               currencyx.Code("USD"),
			ServicePeriod:          servicePeriod,
			InvoiceAt:              servicePeriod.To,
			Price:                  lo.FromPtr(price),
			FeatureKey:             "api-calls",
			ChargeID:               lo.ToPtr("01K00000000000000000000002"),
			ChildUniqueReferenceID: lo.ToPtr("original"),
		},
	}

	result, err := engine.SplitGatheringLine(t.Context(), billing.SplitGatheringLineInput{
		Line:          line,
		FeatureMeters: billingtestFeatureMeters(),
		SplitAt:       splitAt,
	})
	require.NoError(t, err)

	require.Nil(t, result.PreSplitAtLine.SplitLineGroupID)
	require.Nil(t, result.PreSplitAtLine.ChildUniqueReferenceID)
	require.Equal(t, splitAt, result.PreSplitAtLine.ServicePeriod.To)

	require.NotNil(t, result.PostSplitAtLine)
	require.Nil(t, result.PostSplitAtLine.SplitLineGroupID)
	require.Nil(t, result.PostSplitAtLine.ChildUniqueReferenceID)
	require.Equal(t, splitAt, result.PostSplitAtLine.ServicePeriod.From)
}

func billingtestFeatureMeters() feature.FeatureMeters {
	return feature.FeatureMeterCollection{}
}
