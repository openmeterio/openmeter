package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGenerateDetailedLinesRejectsMissingMeteredQuantities(t *testing.T) {
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
			_, err := New(Config{}).GenerateDetailedLines(line)

			// then: rating rejects the input instead of dereferencing a nil quantity
			require.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestGenerateDetailedLinesClampsNegativeMeteredQuantities(t *testing.T) {
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

			result, err := New(Config{}).GenerateDetailedLines(line)

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
}
