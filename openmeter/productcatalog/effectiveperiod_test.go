package productcatalog

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestEffectivePeriod_Validate(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	tests := []struct {
		Name string

		EffectivePeriod          EffectivePeriod
		ExpectedError            bool
		ExpectedValidationIssues models.ValidationIssues
	}{
		{
			Name: "Valid/Zero",
			EffectivePeriod: EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   nil,
			},
			ExpectedError: false,
		},
		{
			Name: "Valid/OpenEnded",
			EffectivePeriod: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(yesterday),
				EffectiveTo:   nil,
			},
			ExpectedError: false,
		},
		{
			Name: "Valid/Range",
			EffectivePeriod: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(yesterday),
				EffectiveTo:   lo.ToPtr(now),
			},
			ExpectedError: false,
		},
		{
			Name: "Invalid/Flipped",
			EffectivePeriod: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
				EffectiveTo:   lo.ToPtr(yesterday),
			},
			ExpectedError: true,
			ExpectedValidationIssues: models.ValidationIssues{
				ErrEffectivePeriodFromAfterTo.WithAttrs(models.Attributes{
					"effectiveFrom": lo.ToPtr(now),
					"effectiveTo":   lo.ToPtr(yesterday),
				}),
			},
		},
		{
			Name: "Invalid/OpenStart",
			EffectivePeriod: EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   lo.ToPtr(now),
			},
			ExpectedError: true,
			ExpectedValidationIssues: models.ValidationIssues{
				ErrEffectivePeriodFromNotSet.WithAttrs(models.Attributes{
					"effectiveFrom": nil,
					"effectiveTo":   lo.ToPtr(now),
				}),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			if test.ExpectedError {
				err := test.EffectivePeriod.Validate()
				assert.Errorf(t, err, "expected invalid effective period")

				issues, err := models.AsValidationIssues(err)
				assert.NoError(t, err)

				models.RequireValidationIssuesMatch(t, test.ExpectedValidationIssues, issues)
			} else {
				assert.NoErrorf(t, test.EffectivePeriod.Validate(), "expected valid effective period")
			}
		})
	}
}

func TestEffectivePeriod_Equal(t *testing.T) {
	tests := []struct {
		Name string

		Left     EffectivePeriod
		Right    EffectivePeriod
		Expected bool
	}{
		{
			Name: "Equal/Nil",
			Left: EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   nil,
			},
			Right: EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   nil,
			},
			Expected: true,
		},
		{
			Name: "Equal/ZeroNil",
			Left: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(time.Time{}),
				EffectiveTo:   lo.ToPtr(time.Time{}),
			},
			Right: EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   nil,
			},
			Expected: true,
		},
		{
			Name: "Valid/Zero",
			Left: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(time.Time{}),
				EffectiveTo:   lo.ToPtr(time.Time{}),
			},
			Right: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(time.Time{}),
				EffectiveTo:   lo.ToPtr(time.Time{}),
			},
			Expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			eq := test.Left.Equal(test.Right)
			assert.Equalf(t, test.Expected, eq, "expected %v, got %v", test.Expected, eq)
		})
	}
}
