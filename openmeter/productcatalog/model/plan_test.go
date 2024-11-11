package model

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestPlanStatus(t *testing.T) {
	now := time.Now()

	tests := []struct {
		Name string

		Effective EffectivePeriod
		Expected  PlanStatus
	}{
		{
			Name: "Draft",
			Effective: EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   nil,
			},
			Expected: DraftStatus,
		},
		{
			Name: "Archived",
			Effective: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now.Add(-1 * time.Hour)),
			},
			Expected: ArchivedStatus,
		},
		{
			Name: "Active with open end",
			Effective: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
				EffectiveTo:   nil,
			},
			Expected: ActiveStatus,
		},
		{
			Name: "Active with fixed end",
			Effective: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now.Add(24 * time.Hour)),
			},
			Expected: ActiveStatus,
		},
		{
			Name: "Scheduled with open end",
			Effective: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
				EffectiveTo:   nil,
			},
			Expected: ScheduledStatus,
		},
		{
			Name: "Scheduled with fixed period",
			Effective: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now.Add(48 * time.Hour)),
			},
			Expected: ScheduledStatus,
		},
		{
			Name: "Invalid with inverse period",
			Effective: EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now.Add(-24 * time.Hour)),
			},
			Expected: InvalidStatus,
		},
		{
			Name: "Invalid with missing start",
			Effective: EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   lo.ToPtr(now.Add(-24 * time.Hour)),
			},
			Expected: InvalidStatus,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, test.Effective.Status())
		})
	}
}
