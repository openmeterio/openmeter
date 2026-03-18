package testutil

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

type TestLineMode string

const (
	SinglePerPeriodLineMode   TestLineMode = "single_per_period"
	MidPeriodSplitLineMode    TestLineMode = "mid_period_split"
	LastInPeriodSplitLineMode TestLineMode = "last_in_period_split"
)

var TestFullPeriod = billing.Period{
	Start: lo.Must(time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")),
	End:   lo.Must(time.Parse(time.RFC3339, "2021-01-02T00:00:00Z")),
}

type FeatureUsageResponse struct {
	LinePeriodQty    alpacadecimal.Decimal
	PreLinePeriodQty alpacadecimal.Decimal
}

type CalculationTestCase struct {
	Price                productcatalog.Price
	Discounts            billing.Discounts
	LineMode             TestLineMode
	Usage                FeatureUsageResponse
	Expect               pricer.DetailedLines
	ExpectErrorIs        error
	PreviousBilledAmount alpacadecimal.Decimal
	CreditsApplied       billing.CreditsApplied
}

type Service interface {
	GenerateDetailedLines(in pricer.StandardLineAccessor) (pricer.GenerateDetailedLinesResult, error)
}

func RunCalculationTestCase(t *testing.T, tc CalculationTestCase) {
	t.Helper()

	line := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				ID:   "fake-line",
				Name: "feature",
			}),
			Currency:          "USD",
			RateCardDiscounts: tc.Discounts,
			CreditsApplied:    tc.CreditsApplied,
		},
		UsageBased: &billing.UsageBasedLine{
			Price: lo.ToPtr(tc.Price),
		},
	}

	fakeParentGroup := billing.SplitLineGroup{
		NamespacedID: models.NamespacedID{
			Namespace: "fake-namespace",
			ID:        "fake-parent-group",
		},
		SplitLineGroupMutableFields: billing.SplitLineGroupMutableFields{
			ServicePeriod: TestFullPeriod,
		},
	}

	fakeHierarchy := billing.SplitLineHierarchy{
		Group: fakeParentGroup,
		Lines: []billing.LineWithInvoiceHeader{
			billing.NewLineWithInvoiceHeader(billing.StandardLineWithInvoiceHeader{
				Line: &billing.StandardLine{
					StandardLineBase: billing.StandardLineBase{
						// Period is unset, so this fake line is always in scope for NetAmount calculations
						Totals: totals.Totals{
							Amount: tc.PreviousBilledAmount,
						},
					},
				},
			}),
		},
	}

	switch tc.LineMode {
	case SinglePerPeriodLineMode:
		line.Period = TestFullPeriod
	case MidPeriodSplitLineMode:
		line.Period = billing.Period{
			Start: TestFullPeriod.Start.Add(time.Hour * 12),
			End:   TestFullPeriod.End.Add(-time.Hour),
		}
		line.SplitLineGroupID = &fakeParentGroup.ID
		line.SplitLineHierarchy = &fakeHierarchy

	case LastInPeriodSplitLineMode:
		line.Period = billing.Period{
			Start: TestFullPeriod.Start.Add(time.Hour * 12),
			End:   TestFullPeriod.End,
		}

		line.SplitLineGroupID = &fakeParentGroup.ID
		line.SplitLineHierarchy = &fakeHierarchy
	}

	// Let's set the usage on the line
	line.UsageBased.Quantity = &tc.Usage.LinePeriodQty
	line.UsageBased.MeteredQuantity = &tc.Usage.LinePeriodQty
	line.UsageBased.PreLinePeriodQuantity = &tc.Usage.PreLinePeriodQty
	line.UsageBased.MeteredPreLinePeriodQuantity = &tc.Usage.PreLinePeriodQty

	service := service.New()

	res, err := service.GenerateDetailedLines(line)
	if err != nil {
		if tc.ExpectErrorIs != nil {
			require.ErrorIs(t, err, tc.ExpectErrorIs)
			return
		}

		t.Fatalf("unexpected error: %v", err)
	}

	if tc.ExpectErrorIs != nil {
		t.Fatalf("expected error: %v", tc.ExpectErrorIs)
	}

	// let's get around nil slices
	if len(tc.Expect) == 0 && len(res.DetailedLines) == 0 {
		return
	}

	expectJSON, err := json.Marshal(tc.Expect)
	require.NoError(t, err)

	resJSON, err := json.Marshal(res.DetailedLines)
	require.NoError(t, err)

	require.JSONEq(t, string(expectJSON), string(resJSON))
}
