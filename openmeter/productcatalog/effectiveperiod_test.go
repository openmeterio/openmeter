package productcatalog

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestEffectivePeriod_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		Name string

		EffectivePeriod EffectivePeriod
		ExpectedError   bool
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
				EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
				EffectiveTo:   nil,
			},
			ExpectedError: false,
		},
		{
			Name: "Valid/Range",
			EffectivePeriod: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now),
			},
			ExpectedError: false,
		},
		{
			Name: "Invalid/Flipped",
			EffectivePeriod: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
				EffectiveTo:   lo.ToPtr(now.Add(-24 * time.Hour)),
			},
			ExpectedError: true,
		},
		{
			Name: "Invalid/OpenStart",
			EffectivePeriod: EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   lo.ToPtr(now),
			},
			ExpectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			if test.ExpectedError {
				assert.Errorf(t, test.EffectivePeriod.Validate(), "expected invalid effective period")
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
