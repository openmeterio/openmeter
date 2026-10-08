package lineengine

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	ratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestRecalculateStandardLineRejectsNonLegacyLinesBeforeSnapshotting(t *testing.T) {
	for _, engineType := range []billing.LineEngineType{
		billing.LineEngineTypeChargeFlatFee,
		billing.LineEngineTypeChargeUsageBased,
		billing.LineEngineTypeChargeCreditPurchase,
	} {
		t.Run(string(engineType), func(t *testing.T) {
			// Given a charge-owned line and no legacy snapshot or rating dependencies.
			line := standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod())
			line.Engine = engineType
			invoice := quantitySnapshotTestInvoice()
			before, err := json.Marshal(line)
			require.NoError(t, err)

			// When the legacy engine is asked to recalculate that line.
			result, err := (&Engine{}).RecalculateStandardLine(t.Context(), RecalculateStandardLineInput{
				Invoice: &invoice,
				Line:    line,
			})

			// Then ownership validation rejects it before any calculation or mutation.
			require.True(t, models.IsGenericValidationError(err))
			require.ErrorContains(t, err, "line must be owned by the legacy invoice engine")
			require.Nil(t, result)
			after, err := json.Marshal(line)
			require.NoError(t, err)
			require.JSONEq(t, string(before), string(after))
		})
	}
}

func TestCollectionRatesReusedAndNewSnapshotsWithoutMutatingInput(t *testing.T) {
	for _, name := range []string{"new snapshot", "completed snapshot", "early force collection"} {
		t.Run(name, func(t *testing.T) {
			// Given a flat line whose prior calculated state is stale.
			line := standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod())
			line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(12),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})
			line.UsageBased.FeatureKey = ""
			line.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
			line.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
			invoice := quantitySnapshotTestInvoice()
			invoice.ID = line.InvoiceID
			invoice.Lines = billing.NewStandardInvoiceLines(billing.StandardLines{line})
			switch name {
			case "completed snapshot":
				invoice.QuantitySnapshotedAt = lo.ToPtr(invoice.DefaultCollectionAtForStandardInvoice())
			case "early force collection":
				invoice.CollectionAt = lo.ToPtr(line.Period.To.Add(time.Hour))
			}

			clock.FreezeTime(line.Period.To)
			defer clock.UnFreeze()
			engine, _ := newQuantitySnapshotTestEngine(t, nil, nil, nil)
			engine.ratingService = ratingservice.New(ratingservice.Config{})
			before, err := json.Marshal(invoice)
			require.NoError(t, err)

			// When collection either snapshots or reuses the existing quantity.
			lines, err := engine.OnCollectionCompleted(t.Context(), billing.OnCollectionCompletedInput{
				Invoice: invoice,
				Lines:   invoice.Lines.OrEmpty(),
			})
			require.NoError(t, err)

			// Then rating completes and only a new snapshot changes raw usage.
			require.Len(t, lines, 1)
			require.Equal(t, float64(12), lines[0].Totals.Total.InexactFloat64())
			require.Len(t, lines[0].DetailedLines, 1)
			if name == "new snapshot" {
				require.Equal(t, float64(1), lines[0].UsageBased.MeteredQuantity.InexactFloat64())
			} else {
				require.Equal(t, float64(7), lines[0].UsageBased.MeteredQuantity.InexactFloat64())
			}

			after, err := json.Marshal(invoice)
			require.NoError(t, err)
			require.JSONEq(t, string(before), string(after))
		})
	}
}

func TestCollectionRatingFailureRetainsOriginalSnapshot(t *testing.T) {
	for _, ratingErr := range []error{billing.ErrInvoiceLineCreditsNotConsumedFully, errors.New("rating unavailable")} {
		t.Run(ratingErr.Error(), func(t *testing.T) {
			// Given prior quantities and a rating failure after a fresh flat snapshot.
			line := standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod())
			line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(12),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			})
			line.UsageBased.FeatureKey = ""
			line.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
			invoice := quantitySnapshotTestInvoice()
			invoice.ID = line.InvoiceID
			invoice.Lines = billing.NewStandardInvoiceLines(billing.StandardLines{line})
			engine, _ := newQuantitySnapshotTestEngine(t, nil, nil, nil)
			rater := newStandardLinesRatingServiceMock(t)
			rater.On("GenerateDetailedLines", mock.Anything, mock.Anything).Return(rating.GenerateDetailedLinesResult{}, ratingErr).Once()
			engine.ratingService = rater
			before, err := json.Marshal(invoice)
			require.NoError(t, err)

			// When rating fails, collection must not leak the attempted quantity.
			_, err = engine.OnCollectionCompleted(t.Context(), billing.OnCollectionCompletedInput{
				Invoice: invoice,
				Lines:   invoice.Lines.OrEmpty(),
			})
			require.ErrorIs(t, err, ratingErr)
			after, err := json.Marshal(invoice)
			require.NoError(t, err)
			require.JSONEq(t, string(before), string(after))
		})
	}
}

func TestCollectionSnapshotFailureRetainsAllOriginalLines(t *testing.T) {
	// Given a flat line that can snapshot and a metered line missing its feature.
	flatLine := standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod())
	flatLine.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount:      alpacadecimal.NewFromInt(12),
		PaymentTerm: productcatalog.InAdvancePaymentTerm,
	})
	flatLine.UsageBased.FeatureKey = ""
	flatLine.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(7))
	missingLine := standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod())
	missingLine.ID = "missing-feature-line"
	invoice := quantitySnapshotTestInvoice()
	invoice.ID = flatLine.InvoiceID
	invoice.Lines = billing.NewStandardInvoiceLines(billing.StandardLines{flatLine, missingLine})
	engine, _ := newQuantitySnapshotTestEngine(t, nil, nil, nil)
	before, err := json.Marshal(invoice)
	require.NoError(t, err)

	// When the batch snapshot fails after another line has successfully snapshotted.
	lines, err := engine.OnCollectionCompleted(t.Context(), billing.OnCollectionCompletedInput{
		Invoice: invoice,
		Lines:   invoice.Lines.OrEmpty(),
	})
	issues, systemErr := billing.ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.Len(t, issues, 1)
	require.Equal(t, billing.ErrInvoiceLineFeatureNotFound.Code, issues[0].Code)

	// Then no replacement or in-place quantity changes escape the failed attempt.
	require.Nil(t, lines)
	after, err := json.Marshal(invoice)
	require.NoError(t, err)
	require.JSONEq(t, string(before), string(after))
}

func TestRateStandardLinesUsesMeteredUsageAndPreservesIdentities(t *testing.T) {
	// Given raw usage, stale rated quantities, and discounts without correlation IDs.
	line := standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod())
	line.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.RequireFromString("2.345"),
	})
	line.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(10))
	line.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
	line.UsageBased.Quantity = lo.ToPtr(alpacadecimal.NewFromInt(1))
	line.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.NewFromInt(999))
	line.RateCardDiscounts = billing.Discounts{
		Usage: &billing.UsageDiscount{
			UsageDiscount: productcatalog.UsageDiscount{Quantity: alpacadecimal.NewFromInt(2)},
		},
		Percentage: &billing.PercentageDiscount{
			PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(25)},
		},
	}
	engine := &Engine{ratingService: ratingservice.New(ratingservice.Config{})}
	input := billing.StandardLines{line}
	before, err := json.Marshal(input)
	require.NoError(t, err)

	// When the supplied snapshots are rated without an invoice or persistence dependency.
	lines, err := engine.RateStandardLines(input)
	require.NoError(t, err)

	// Then normalization and rating affect only the result, using raw rather than net usage.
	after, err := json.Marshal(input)
	require.NoError(t, err)
	require.JSONEq(t, string(before), string(after))
	require.Len(t, lines, 1)
	rated := lines[0]
	require.NotSame(t, line, rated)
	require.NotEmpty(t, rated.RateCardDiscounts.Usage.CorrelationID)
	require.NotEmpty(t, rated.RateCardDiscounts.Percentage.CorrelationID)
	require.Equal(t, float64(10), rated.UsageBased.MeteredQuantity.InexactFloat64())
	require.Equal(t, float64(8), rated.UsageBased.Quantity.InexactFloat64())
	require.Zero(t, rated.UsageBased.PreLinePeriodQuantity.InexactFloat64())
	require.Equal(t, 18.76, rated.Totals.Amount.InexactFloat64())
	require.Equal(t, 14.07, rated.Totals.Total.InexactFloat64())
	require.Len(t, rated.Discounts.Usage, 1)
	require.Equal(t, float64(2), rated.Discounts.Usage[0].Quantity.InexactFloat64())
	require.Len(t, rated.DetailedLines, 1)
	require.Len(t, rated.DetailedLines[0].AmountDiscounts, 1)
	require.Equal(t, 4.69, rated.DetailedLines[0].AmountDiscounts[0].Amount.InexactFloat64())

	// Given persisted child and amount-discount identities and an edited price.
	rated.DetailedLines[0].ID = "detail-id"
	rated.DetailedLines[0].FeeLineConfigID = "detail-config-id"
	rated.DetailedLines[0].CreatedAt = line.Period.From
	rated.DetailedLines[0].UpdatedAt = line.Period.To
	rated.DetailedLines[0].AmountDiscounts[0].ManagedModelWithID = models.ManagedModelWithID{
		ID: "amount-discount-id",
		ManagedModel: models.ManagedModel{
			CreatedAt: line.Period.From,
			UpdatedAt: line.Period.To,
		},
	}
	rated.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.RequireFromString("3.125"),
	})
	input = lines
	before, err = json.Marshal(input)
	require.NoError(t, err)

	// When the edited state is rated again.
	rerated, err := engine.RateStandardLines(input)
	require.NoError(t, err)

	// Then discounts are applied once and matching persisted identities survive.
	after, err = json.Marshal(input)
	require.NoError(t, err)
	require.JSONEq(t, string(before), string(after))
	require.Equal(t, rated.RateCardDiscounts, rerated[0].RateCardDiscounts)
	require.Equal(t, float64(8), rerated[0].UsageBased.Quantity.InexactFloat64())
	require.Equal(t, float64(25), rerated[0].Totals.Amount.InexactFloat64())
	require.Equal(t, 18.75, rerated[0].Totals.Total.InexactFloat64())
	require.Len(t, rerated[0].Discounts.Usage, 1)
	require.Equal(t, float64(2), rerated[0].Discounts.Usage[0].Quantity.InexactFloat64())
	require.Len(t, rerated[0].DetailedLines, 1)
	detail := rerated[0].DetailedLines[0]
	require.Equal(t, rated.DetailedLines[0].ManagedResource, detail.ManagedResource)
	require.Equal(t, "detail-config-id", detail.FeeLineConfigID)
	require.Equal(t, 0, lo.FromPtr(detail.Index))
	require.Len(t, detail.AmountDiscounts, 1)
	require.Equal(t, "amount-discount-id", detail.AmountDiscounts[0].ID)
	require.Equal(t, line.Period.From, detail.AmountDiscounts[0].CreatedAt)
	require.True(t, detail.AmountDiscounts[0].UpdatedAt.IsZero()) // The adapter owns the update timestamp.
	require.Equal(t, 6.25, detail.AmountDiscounts[0].Amount.InexactFloat64())

	repeated, err := engine.RateStandardLines(rerated)
	require.NoError(t, err)
	require.Equal(t, rerated, repeated)
}

func TestRateStandardLinesFailuresLeaveTheWholeInputUnchanged(t *testing.T) {
	for _, tc := range []struct {
		name       string
		invalidate func(*billing.StandardLine)
		errorText  string
	}{
		{
			name: "rating failure",
			invalidate: func(line *billing.StandardLine) {
				line.UsageBased.MeteredQuantity = nil
			},
			errorText: "metered quantity is required",
		},
		{
			name: "mapping failure",
			invalidate: func(line *billing.StandardLine) {
				line.InvoiceID = ""
			},
			errorText: "merging generated detailed lines for line[second-line]",
		},
		{
			name: "output validation failure",
			invalidate: func(line *billing.StandardLine) {
				line.ManagedBy = "invalid"
			},
			errorText: "validating standard line[second-line]",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a valid first line and a second line that fails after rating starts.
			lines := billing.StandardLines{
				standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod()),
				standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod()),
			}
			lines[1].ID = "second-line"
			for _, line := range lines {
				line.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(10))
				line.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
				line.RateCardDiscounts = usageDiscountForLineEngineOverrideTest("2")
			}

			tc.invalidate(lines[1])
			before, err := json.Marshal(lines)
			require.NoError(t, err)

			// When rating the batch fails on the second line.
			engine := &Engine{ratingService: ratingservice.New(ratingservice.Config{})}
			out, err := engine.RateStandardLines(lines)

			// Then neither normalization nor earlier successful rating leaks into the input.
			require.ErrorContains(t, err, tc.errorText)
			require.Nil(t, out)
			after, err := json.Marshal(lines)
			require.NoError(t, err)
			require.JSONEq(t, string(before), string(after))
		})
	}
}

func TestRateStandardLinesPreservesCalculatedResultsWithCriticalIssues(t *testing.T) {
	// Given a rating result accompanied by a typed critical issue.
	line := standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod())
	result := rating.GenerateDetailedLinesResult{
		FinalUsage: &rating.Usage{Quantity: alpacadecimal.NewFromInt(2)},
	}

	// When rating records issues through the existing all-severity contract.
	ratingService := newStandardLinesRatingServiceMock(t)
	ratingService.On("GenerateDetailedLines", mock.Anything, mock.Anything).
		Return(result, billing.ErrInvoiceLineVolumeSplitNotSupported).Once()
	engine := &Engine{ratingService: ratingService}
	lines, err := engine.RateStandardLines(billing.StandardLines{line})

	// Then calculated state is returned with the critical issue and line attribution.
	issues, systemErr := billing.ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.Len(t, issues, 1)
	require.Equal(t, billing.ValidationIssueSeverityCritical, issues[0].Severity)
	require.Equal(t, billing.ErrInvoiceLineVolumeSplitNotSupported.Code, issues[0].Code)
	require.Equal(t, line.ID, issues[0].Attributes[billing.AttributeKeyLineID])
	require.Len(t, lines, 1)
	require.Equal(t, float64(2), lines[0].UsageBased.Quantity.InexactFloat64())
	require.Nil(t, line.UsageBased.Quantity)
}

func TestRateStandardLinesPreservesResultsWithValidationWarnings(t *testing.T) {
	line := standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod())
	ratingResult := rating.GenerateDetailedLinesResult{
		FinalUsage: &rating.Usage{
			Quantity:              alpacadecimal.NewFromInt(2),
			PreLinePeriodQuantity: alpacadecimal.Zero,
		},
		Totals: totals.Totals{
			Amount: alpacadecimal.NewFromInt(2),
			Total:  alpacadecimal.NewFromInt(2),
		},
	}
	ratingService := newStandardLinesRatingServiceMock(t)
	ratingService.On("GenerateDetailedLines", mock.Anything, mock.Anything).
		Return(ratingResult, billing.ValidationWithComponent(
			billing.ValidationComponentBillingRating,
			billing.WarnNegativeMeteredQuantityClamped,
		)).Once()
	engine := &Engine{ratingService: ratingService}

	lines, err := engine.RateStandardLines(billing.StandardLines{line})

	issues, systemErr := billing.ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.Equal(t, billing.ValidationIssues{{
		Severity:  billing.ValidationIssueSeverityWarning,
		Message:   billing.WarnNegativeMeteredQuantityClamped.Message,
		Code:      billing.WarnNegativeMeteredQuantityClamped.Code,
		Component: billing.ValidationComponentBillingRating,
		Attributes: models.Annotations{
			billing.AttributeKeyLineID: line.ID,
		},
	}}, issues)
	require.Len(t, lines, 1)
	require.NotSame(t, line, lines[0])
	require.Equal(t, line.ID, lines[0].ID)
	require.Equal(t, float64(2), lines[0].UsageBased.Quantity.InexactFloat64())
	require.Equal(t, float64(2), lines[0].Totals.Amount.InexactFloat64())
	require.Equal(t, float64(2), lines[0].Totals.Total.InexactFloat64())
	require.Nil(t, line.UsageBased.Quantity)
	require.Zero(t, line.Totals.Total.InexactFloat64())
}

func TestRateStandardLinesRejectsSystemErrors(t *testing.T) {
	ratingErr := errors.New("rating unavailable")
	ratingService := newStandardLinesRatingServiceMock(t)
	ratingService.On("GenerateDetailedLines", mock.Anything, mock.Anything).
		Return(rating.GenerateDetailedLinesResult{}, ratingErr).Once()
	engine := &Engine{ratingService: ratingService}

	lines, err := engine.RateStandardLines(billing.StandardLines{
		standardLineForLineEngineOverrideTest(t, lineEngineOverrideTestPeriod()),
	})

	require.ErrorIs(t, err, ratingErr)
	require.Nil(t, lines)
}

type standardLinesRatingServiceMock struct {
	mock.Mock
}

func newStandardLinesRatingServiceMock(t *testing.T) *standardLinesRatingServiceMock {
	t.Helper()

	service := &standardLinesRatingServiceMock{}
	service.Test(t)
	t.Cleanup(func() {
		service.AssertExpectations(t)
	})

	return service
}

func (s *standardLinesRatingServiceMock) GenerateDetailedLines(in rating.StandardLineAccessor, opts ...rating.GenerateDetailedLinesOption) (rating.GenerateDetailedLinesResult, error) {
	args := s.Called(in, opts)
	return args.Get(0).(rating.GenerateDetailedLinesResult), args.Error(1)
}

func (s *standardLinesRatingServiceMock) ResolveBillablePeriod(in rating.ResolveBillablePeriodInput) (billing.IsLineBillableAsOfResult, error) {
	args := s.Called(in)
	return args.Get(0).(billing.IsLineBillableAsOfResult), args.Error(1)
}
