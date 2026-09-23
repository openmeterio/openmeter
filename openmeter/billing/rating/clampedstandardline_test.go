package rating

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestClampedStandardLineAccessor(t *testing.T) {
	tests := []struct {
		name                          string
		meteredQuantity               alpacadecimal.Decimal
		meteredPreLinePeriodQuantity  alpacadecimal.Decimal
		expectedMeteredQuantity       alpacadecimal.Decimal
		expectedPreLinePeriodQuantity alpacadecimal.Decimal
		expectedWarnings              []error
	}{
		{
			name:                          "positive quantities",
			meteredQuantity:               alpacadecimal.NewFromInt(2),
			meteredPreLinePeriodQuantity:  alpacadecimal.NewFromInt(1),
			expectedMeteredQuantity:       alpacadecimal.NewFromInt(2),
			expectedPreLinePeriodQuantity: alpacadecimal.NewFromInt(1),
		},
		{
			name:                          "zero quantities",
			meteredQuantity:               alpacadecimal.Zero,
			meteredPreLinePeriodQuantity:  alpacadecimal.Zero,
			expectedMeteredQuantity:       alpacadecimal.Zero,
			expectedPreLinePeriodQuantity: alpacadecimal.Zero,
		},
		{
			name:                          "negative current quantity",
			meteredQuantity:               alpacadecimal.NewFromInt(-2),
			meteredPreLinePeriodQuantity:  alpacadecimal.NewFromInt(1),
			expectedMeteredQuantity:       alpacadecimal.Zero,
			expectedPreLinePeriodQuantity: alpacadecimal.NewFromInt(1),
			expectedWarnings:              []error{billing.WarnNegativeMeteredQuantityClamped},
		},
		{
			name:                          "negative pre-line quantity",
			meteredQuantity:               alpacadecimal.NewFromInt(2),
			meteredPreLinePeriodQuantity:  alpacadecimal.NewFromInt(-1),
			expectedMeteredQuantity:       alpacadecimal.NewFromInt(2),
			expectedPreLinePeriodQuantity: alpacadecimal.Zero,
			expectedWarnings:              []error{billing.WarnNegativePreLinePeriodMeteredQuantityClamped},
		},
		{
			name:                          "both quantities negative",
			meteredQuantity:               alpacadecimal.NewFromInt(-2),
			meteredPreLinePeriodQuantity:  alpacadecimal.NewFromInt(-1),
			expectedMeteredQuantity:       alpacadecimal.Zero,
			expectedPreLinePeriodQuantity: alpacadecimal.Zero,
			expectedWarnings: []error{
				billing.WarnNegativeMeteredQuantityClamped,
				billing.WarnNegativePreLinePeriodMeteredQuantityClamped,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			line := &billing.StandardLine{
				UsageBased: &billing.UsageBasedLine{
					MeteredQuantity:              &tc.meteredQuantity,
					MeteredPreLinePeriodQuantity: &tc.meteredPreLinePeriodQuantity,
				},
			}
			accessor, err := NewClampedStandardLineAccessor(line)
			require.NoError(t, err)

			meteredQuantity, err := accessor.GetMeteredQuantity()
			require.NoError(t, err)
			meteredPreLinePeriodQuantity, err := accessor.GetMeteredPreLinePeriodQuantity()
			require.NoError(t, err)

			require.Equal(t, tc.expectedMeteredQuantity, *meteredQuantity)
			require.Equal(t, tc.expectedPreLinePeriodQuantity, *meteredPreLinePeriodQuantity)
			require.Equal(t, tc.meteredQuantity, *line.UsageBased.MeteredQuantity)
			require.Equal(t, tc.meteredPreLinePeriodQuantity, *line.UsageBased.MeteredPreLinePeriodQuantity)
			if len(tc.expectedWarnings) > 0 {
				warnings := accessor.GetValidationWarnings()
				for _, expectedWarning := range tc.expectedWarnings {
					require.ErrorIs(t, warnings, expectedWarning)
				}
				issues, systemErr := billing.ToValidationIssues(warnings)
				require.NoError(t, systemErr)
				require.Len(t, issues, len(tc.expectedWarnings))
				for _, issue := range issues {
					require.Equal(t, billing.ValidationComponentBillingRating, issue.Component)
					require.Equal(t, models.Annotations{
						"original_metered_quantity":          tc.meteredQuantity.String(),
						"original_pre_line_metered_quantity": tc.meteredPreLinePeriodQuantity.String(),
					}, issue.Attributes)
				}
			} else {
				require.NoError(t, accessor.GetValidationWarnings())
			}
		})
	}
}

func TestClampedStandardLineAccessorWarningOmitsNilQuantities(t *testing.T) {
	negativeQuantity := alpacadecimal.NewFromInt(-1)

	for _, tc := range []struct {
		name                         string
		meteredQuantity              *alpacadecimal.Decimal
		meteredPreLinePeriodQuantity *alpacadecimal.Decimal
		expectedAttributes           models.Annotations
	}{
		{
			name:            "pre-line quantity is nil",
			meteredQuantity: &negativeQuantity,
			expectedAttributes: models.Annotations{
				"original_metered_quantity": "-1",
			},
		},
		{
			name:                         "current quantity is nil",
			meteredPreLinePeriodQuantity: &negativeQuantity,
			expectedAttributes: models.Annotations{
				"original_pre_line_metered_quantity": "-1",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			accessor, err := NewClampedStandardLineAccessor(&billing.StandardLine{
				UsageBased: &billing.UsageBasedLine{
					MeteredQuantity:              tc.meteredQuantity,
					MeteredPreLinePeriodQuantity: tc.meteredPreLinePeriodQuantity,
				},
			})
			require.NoError(t, err)

			issues, systemErr := billing.ToValidationIssues(accessor.GetValidationWarnings())
			require.NoError(t, systemErr)
			require.Len(t, issues, 1)
			require.Equal(t, tc.expectedAttributes, issues[0].Attributes)
		})
	}
}

func TestClampedStandardLineAccessorPreservesNilAndErrors(t *testing.T) {
	t.Run("nil quantities", func(t *testing.T) {
		accessor, err := NewClampedStandardLineAccessor(&billing.StandardLine{UsageBased: &billing.UsageBasedLine{}})
		require.NoError(t, err)

		quantity, err := accessor.GetMeteredQuantity()
		require.NoError(t, err)
		require.Nil(t, quantity)
		preLinePeriodQuantity, err := accessor.GetMeteredPreLinePeriodQuantity()
		require.NoError(t, err)
		require.Nil(t, preLinePeriodQuantity)
		require.NoError(t, accessor.GetValidationWarnings())
	})

	t.Run("accessor errors", func(t *testing.T) {
		_, err := NewClampedStandardLineAccessor(&billing.StandardLine{})
		require.EqualError(t, err, "usage based line is required")
	})
}
