package service

import (
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGenerateLegacyProgressiveBilledLinesRejectsMissingMeteredQuantities(t *testing.T) {
	quantity := alpacadecimal.NewFromInt(1)

	for _, tc := range []struct {
		name                         string
		meteredQuantity              *alpacadecimal.Decimal
		meteredPreLinePeriodQuantity *alpacadecimal.Decimal
		expectedError                string
	}{
		{
			name:                         "current quantity",
			meteredPreLinePeriodQuantity: &quantity,
			expectedError:                "metered quantity is required",
		},
		{
			name:            "pre-line period quantity",
			meteredQuantity: &quantity,
			expectedError:   "pre-line period metered quantity is required",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: a metered line whose quantity snapshot is incomplete
			line := &billing.StandardLine{
				StandardLineBase: billing.StandardLineBase{
					Currency: "USD",
					Period: timeutil.ClosedPeriod{
						From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						To:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
					},
				},
				UsageBased: &billing.UsageBasedLine{
					Price:                        productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: quantity}),
					MeteredQuantity:              tc.meteredQuantity,
					MeteredPreLinePeriodQuantity: tc.meteredPreLinePeriodQuantity,
				},
			}

			// when: detailed-line calculation is invoked despite the incomplete snapshot
			_, err := New(Config{}).GenerateLegacyProgressiveBilledLines(legacyRatingTestLine{StandardLine: line})

			// then: rating rejects the input instead of dereferencing a nil quantity
			require.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestRatingReturnsAmountsBeforeCreditApplication(t *testing.T) {
	for _, price := range []*productcatalog.Price{
		productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(10)}),
		productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(100),
			PaymentTerm: productcatalog.InArrearsPaymentTerm,
		}),
	} {
		t.Run(string(price.Type()), func(t *testing.T) {
			for _, mode := range []struct {
				name string
				rate func(rating.Service, *billing.StandardLine) (rating.GenerateDetailedLinesResult, error)
			}{
				{
					name: "ordinary",
					rate: func(svc rating.Service, line *billing.StandardLine) (rating.GenerateDetailedLinesResult, error) {
						return svc.GenerateDetailedLines(line)
					},
				},
				{
					name: "legacy",
					rate: func(svc rating.Service, line *billing.StandardLine) (rating.GenerateDetailedLinesResult, error) {
						return svc.GenerateLegacyProgressiveBilledLines(legacyRatingTestLine{StandardLine: line})
					},
				},
			} {
				t.Run(mode.name, func(t *testing.T) {
					// Given pricing worth 100 and invoice credit facts owned outside rating.
					line := &billing.StandardLine{
						StandardLineBase: billing.StandardLineBase{
							Currency: "USD",
							Period: timeutil.ClosedPeriod{
								From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
								To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
							},
							CreditsApplied: billing.CreditsApplied{{Amount: alpacadecimal.NewFromInt(30)}},
						},
						UsageBased: &billing.UsageBasedLine{
							Price:                        price.Clone(),
							MeteredQuantity:              lo.ToPtr(alpacadecimal.NewFromInt(10)),
							MeteredPreLinePeriodQuantity: lo.ToPtr(alpacadecimal.Zero),
						},
					}
					svc := New(Config{})

					// When either rating entrypoint calculates the line.
					result, err := mode.rate(svc, line)
					require.NoError(t, err)

					// Then rating leaves credit application to the owner of those facts.
					require.Equal(t, float64(100), result.Totals.Total.InexactFloat64())
					require.Zero(t, result.Totals.CreditsTotal.InexactFloat64())
					require.Len(t, result.DetailedLines, 1)
					require.Equal(t, float64(100), result.DetailedLines[0].Totals.Total.InexactFloat64())
					require.Equal(t, float64(30), line.CreditsApplied[0].Amount.InexactFloat64())
				})
			}
		})
	}
}

func TestGenerateLegacyProgressiveBilledLinesClampsNegativeMeteredQuantities(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		meteredQuantity                alpacadecimal.Decimal
		meteredPreLinePeriodQuantity   alpacadecimal.Decimal
		expectedFinalQuantity          alpacadecimal.Decimal
		expectedFinalPrePeriodQuantity alpacadecimal.Decimal
		expectedWarning                billing.ValidationIssue
	}{
		{
			name:                           "current quantity",
			meteredQuantity:                alpacadecimal.NewFromInt(-1),
			meteredPreLinePeriodQuantity:   alpacadecimal.Zero,
			expectedFinalQuantity:          alpacadecimal.Zero,
			expectedFinalPrePeriodQuantity: alpacadecimal.Zero,
			expectedWarning:                billing.WarnNegativeMeteredQuantityClamped,
		},
		{
			name:                           "pre-line period quantity",
			meteredQuantity:                alpacadecimal.Zero,
			meteredPreLinePeriodQuantity:   alpacadecimal.NewFromInt(-1),
			expectedFinalQuantity:          alpacadecimal.Zero,
			expectedFinalPrePeriodQuantity: alpacadecimal.Zero,
			expectedWarning:                billing.WarnNegativePreLinePeriodMeteredQuantityClamped,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, mode := range []struct {
				name  string
				setup func(*legacyRatingTestLine)
			}{
				{
					name:  "unsplit",
					setup: func(*legacyRatingTestLine) {},
				},
				{
					name: "progressive",
					setup: func(line *legacyRatingTestLine) {
						line.SplitLineGroupID = lo.ToPtr("group-id")
						line.fullServicePeriod = &line.Period
					},
				},
			} {
				t.Run(mode.name, func(t *testing.T) {
					// Given negative raw usage on an unsplit or progressively billed legacy line.
					line := &billing.StandardLine{
						StandardLineBase: billing.StandardLineBase{
							Currency: "USD",
							Period: timeutil.ClosedPeriod{
								From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
								To:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
							},
						},
						UsageBased: &billing.UsageBasedLine{
							Price:                        productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
							MeteredQuantity:              &tc.meteredQuantity,
							MeteredPreLinePeriodQuantity: &tc.meteredPreLinePeriodQuantity,
						},
					}

					legacyLine := legacyRatingTestLine{StandardLine: line}
					mode.setup(&legacyLine)

					// When rating clamps the quantities without changing the source snapshot.
					result, err := New(Config{}).GenerateLegacyProgressiveBilledLines(legacyLine)

					// Then usable zero totals accompany the same attributed warning in both modes.
					require.ErrorIs(t, err, tc.expectedWarning)
					issues, systemErr := billing.ToValidationIssues(err)
					require.NoError(t, systemErr)
					require.Equal(t, billing.ValidationIssues{{
						Severity:  billing.ValidationIssueSeverityWarning,
						Message:   tc.expectedWarning.Message,
						Code:      tc.expectedWarning.Code,
						Component: billing.ValidationComponentBillingRating,
						Attributes: models.Annotations{
							"original_metered_quantity":          tc.meteredQuantity.String(),
							"original_pre_line_metered_quantity": tc.meteredPreLinePeriodQuantity.String(),
						},
					}}, issues)
					require.Empty(t, result.DetailedLines)
					require.NotNil(t, result.FinalUsage)
					require.Equal(t, tc.expectedFinalQuantity, result.FinalUsage.Quantity)
					require.Equal(t, tc.expectedFinalPrePeriodQuantity, result.FinalUsage.PreLinePeriodQuantity)
					require.True(t, result.Totals.Total.IsZero())
					require.Equal(t, tc.meteredQuantity, *line.UsageBased.MeteredQuantity)
					require.Equal(t, tc.meteredPreLinePeriodQuantity, *line.UsageBased.MeteredPreLinePeriodQuantity)
				})
			}
		})
	}
}

func TestGenerateDetailedLinesDoesNotReadLegacyHistory(t *testing.T) {
	for _, tc := range []struct {
		name             string
		quantity         int64
		expectedQuantity float64
		expectedCharges  float64
	}{
		{name: "positive usage", quantity: 20, expectedQuantity: 20, expectedCharges: 80},
		{name: "negative usage", quantity: -1, expectedQuantity: 0, expectedCharges: 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given ordinary rating data with a stale split ID and unusable legacy history.
			line := &billing.StandardLine{
				StandardLineBase: billing.StandardLineBase{
					Currency:         "USD",
					SplitLineGroupID: lo.ToPtr("unexpanded-group"),
					Period: timeutil.ClosedPeriod{
						From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
						To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
					},
				},
				UsageBased: &billing.UsageBasedLine{
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
						Commitments: productcatalog.Commitments{
							MinimumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
						},
					}),
					MeteredQuantity:              lo.ToPtr(alpacadecimal.NewFromInt(tc.quantity)),
					MeteredPreLinePeriodQuantity: lo.ToPtr(alpacadecimal.NewFromInt(-10)),
				},
			}
			svc := New(Config{})

			// When the ordinary entrypoint rates the current quantity.
			result, err := svc.GenerateDetailedLines(line)

			// Then prior usage and split history are ignored, and current-usage warnings survive.
			if tc.quantity < 0 {
				require.ErrorIs(t, err, billing.WarnNegativeMeteredQuantityClamped)
				issues, systemErr := billing.ToValidationIssues(err)
				require.NoError(t, systemErr)
				require.Len(t, issues, 1)
				require.Equal(t, models.Annotations{
					"original_metered_quantity":          "-1",
					"original_pre_line_metered_quantity": "0",
				}, issues[0].Attributes)
			} else {
				require.NoError(t, err)
			}

			require.NotNil(t, result.FinalUsage)
			require.Equal(t, tc.expectedQuantity, result.FinalUsage.Quantity.InexactFloat64())
			require.Zero(t, result.FinalUsage.PreLinePeriodQuantity.InexactFloat64())
			require.Equal(t, tc.expectedCharges, result.Totals.ChargesTotal.InexactFloat64())
			require.Equal(t, float64(100), result.Totals.Total.InexactFloat64())
			require.Equal(t, float64(tc.quantity), line.UsageBased.MeteredQuantity.InexactFloat64())
			require.Equal(t, float64(-10), line.UsageBased.MeteredPreLinePeriodQuantity.InexactFloat64())

			// The legacy entrypoint retains its strict missing-hierarchy failure.
			_, err = svc.GenerateLegacyProgressiveBilledLines(legacyRatingTestLine{StandardLine: line})
			require.ErrorContains(t, err, "split line hierarchy is required")
		})
	}
}

// legacyRatingTestLine supplies the legacy accessor contract independently of
// ordinary invoice lines; persisted history resolution is tested by the engine.
type legacyRatingTestLine struct {
	*billing.StandardLine

	fullServicePeriod *timeutil.ClosedPeriod
}

func (l legacyRatingTestLine) GetProgressivelyBilledServicePeriod() (timeutil.ClosedPeriod, error) {
	if !l.IsProgressivelyBilled() {
		return l.Period, nil
	}

	if l.fullServicePeriod == nil {
		return timeutil.ClosedPeriod{}, errors.New("split line hierarchy is required")
	}

	return *l.fullServicePeriod, nil
}

func (l legacyRatingTestLine) GetPreviouslyBilledAmount() (alpacadecimal.Decimal, error) {
	return alpacadecimal.Zero, nil
}
