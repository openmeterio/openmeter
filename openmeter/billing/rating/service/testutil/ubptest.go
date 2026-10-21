package testutil

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	stdinvoicesanity "github.com/openmeterio/openmeter/openmeter/billing/stdinvoice/sanity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type TestLineMode string

const (
	SinglePerPeriodLineMode   TestLineMode = "single_per_period"
	MidPeriodSplitLineMode    TestLineMode = "mid_period_split"
	LastInPeriodSplitLineMode TestLineMode = "last_in_period_split"
)

var TestFullPeriod = timeutil.ClosedPeriod{
	From: lo.Must(time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")),
	To:   lo.Must(time.Parse(time.RFC3339, "2021-01-02T00:00:00Z")),
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
	Expect               rating.DetailedLines
	ExpectErrorIs        error
	PreviousBilledAmount alpacadecimal.Decimal
	Options              []rating.GenerateDetailedLinesOption

	// UnitConfig, when set, is exposed on the rated line via GetUnitConfig. Standard
	// invoice lines do not carry a unit_config in production (only the charges path
	// does), so the harness injects it through a wrapper accessor for testing.
	UnitConfig *productcatalog.UnitConfig
	// UnitConfigEnabled toggles the deploy-wide unitConfig.enabled flag on the rating
	// service under test.
	UnitConfigEnabled bool
}

// testLineAccessor supplies rating context without constructing an invoice hierarchy.
type testLineAccessor struct {
	*billing.StandardLine

	unitConfig             *productcatalog.UnitConfig
	progressivelyBilled    bool
	previouslyBilledAmount alpacadecimal.Decimal
}

func (l testLineAccessor) GetUnitConfig() *productcatalog.UnitConfig {
	return l.unitConfig
}

type Service interface {
	GenerateLegacyProgressiveBilledLines(in rating.LegacyProgressivelyBillableLineAccessor, opts ...rating.GenerateDetailedLinesOption) (rating.GenerateDetailedLinesResult, error)
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
		},
		UsageBased: &billing.UsageBasedLine{
			Price: lo.ToPtr(tc.Price),
		},
	}

	switch tc.LineMode {
	case SinglePerPeriodLineMode:
		line.Period = TestFullPeriod
	case MidPeriodSplitLineMode:
		line.Period = timeutil.ClosedPeriod{
			From: TestFullPeriod.From.Add(time.Hour * 12),
			To:   TestFullPeriod.To.Add(-time.Hour),
		}

	case LastInPeriodSplitLineMode:
		line.Period = timeutil.ClosedPeriod{
			From: TestFullPeriod.From.Add(time.Hour * 12),
			To:   TestFullPeriod.To,
		}

	}

	// Let's set the usage on the line
	line.UsageBased.Quantity = &tc.Usage.LinePeriodQty
	line.UsageBased.MeteredQuantity = &tc.Usage.LinePeriodQty
	line.UsageBased.PreLinePeriodQuantity = &tc.Usage.PreLinePeriodQty
	line.UsageBased.MeteredPreLinePeriodQuantity = &tc.Usage.PreLinePeriodQty

	accessor := testLineAccessor{
		StandardLine:        line,
		unitConfig:          tc.UnitConfig,
		progressivelyBilled: tc.LineMode != SinglePerPeriodLineMode,
	}
	if accessor.progressivelyBilled {
		accessor.previouslyBilledAmount = tc.PreviousBilledAmount
	}

	svc := service.New(service.Config{UnitConfigEnabled: tc.UnitConfigEnabled})

	res, err := svc.GenerateLegacyProgressiveBilledLines(accessor, tc.Options...)
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

	// Add the IDs and invoice fields that rating does not create, so we can
	// check its output as an app would receive it. Keep the calculated amounts
	// unchanged so these checks can catch mistakes in rating.
	snapshot := &billing.StandardLine{
		ID:             "line-1",
		Namespace:      "sanity-rating",
		InvoiceID:      "invoice-1",
		Currency:       line.Currency,
		Period:         line.Period,
		UsageBased:     &billing.UsageBasedLine{Price: line.UsageBased.Price},
		CreditsApplied: append(billing.CreditsApplied{}, line.CreditsApplied...),
		Totals:         res.Totals,
	}
	for idx := range snapshot.CreditsApplied {
		if snapshot.CreditsApplied[idx].CreditRealizationID == "" {
			snapshot.CreditsApplied[idx].CreditRealizationID = "rating-realization"
		}
	}

	snapshot.DetailedLines = lo.Map(res.DetailedLines, func(detail rating.DetailedLine, idx int) billing.DetailedLine {
		mapped := billing.DetailedLine{
			ID:                     fmt.Sprintf("detail-%d", idx),
			Namespace:              snapshot.Namespace,
			InvoiceID:              snapshot.InvoiceID,
			ChildUniqueReferenceID: detail.ChildUniqueReferenceID,
			ServicePeriod:          lo.FromPtrOr(detail.Period, snapshot.Period),
			PerUnitAmount:          detail.PerUnitAmount,
			Quantity:               detail.Quantity,
			Category:               lo.CoalesceOrEmpty(detail.Category, stddetailedline.CategoryRegular),
			AmountDiscounts:        detail.AmountDiscounts.Clone(),
			CreditsApplied:         append(billing.CreditsApplied{}, detail.CreditsApplied...),
			Totals:                 detail.Totals,
		}
		for j := range mapped.AmountDiscounts {
			mapped.AmountDiscounts[j].ID = fmt.Sprintf("discount-%d-%d", idx, j)
		}

		for j := range mapped.CreditsApplied {
			if mapped.CreditsApplied[j].CreditRealizationID == "" {
				mapped.CreditsApplied[j].CreditRealizationID = "rating-realization"
			}
		}

		return mapped
	})
	sanityErr := stdinvoicesanity.Validate(billing.StandardInvoice{
		ID:        snapshot.InvoiceID,
		Namespace: snapshot.Namespace,
		Period:    lo.ToPtr(snapshot.Period),
		Currency:  snapshot.Currency,
		Totals:    res.Totals,
		Lines:     billing.NewStandardInvoiceLines(billing.StandardLines{snapshot}),
	})
	options := rating.NewGenerateDetailedLinesOptions(tc.Options...)
	if options.DisableCreditsMutator && len(snapshot.CreditsApplied) > 0 {
		// Credits were disabled for this rating step, so the details are missing
		// the parent's allocations and should fail the check.
		require.ErrorIs(t, sanityErr, stdinvoicesanity.ErrCreditConservationMismatch)
	} else {
		require.NoError(t, sanityErr)
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

func (l testLineAccessor) IsProgressivelyBilled() bool {
	return l.progressivelyBilled
}

func (l testLineAccessor) GetProgressivelyBilledServicePeriod() (timeutil.ClosedPeriod, error) {
	return TestFullPeriod, nil
}

func (l testLineAccessor) GetPreviouslyBilledAmount() (alpacadecimal.Decimal, error) {
	return l.previouslyBilledAmount, nil
}
