package productcatalog_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestPlanStatus(t *testing.T) {
	now := time.Now()

	tests := []struct {
		Name string

		Effective productcatalog.EffectivePeriod
		Expected  productcatalog.PlanStatus
	}{
		{
			Name: "Draft",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   nil,
			},
			Expected: productcatalog.DraftStatus,
		},
		{
			Name: "Archived",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now.Add(-1 * time.Hour)),
			},
			Expected: productcatalog.ArchivedStatus,
		},
		{
			Name: "Active with open end",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
				EffectiveTo:   nil,
			},
			Expected: productcatalog.ActiveStatus,
		},
		{
			Name: "Active with fixed end",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(-24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now.Add(24 * time.Hour)),
			},
			Expected: productcatalog.ActiveStatus,
		},
		{
			Name: "Scheduled with open end",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
				EffectiveTo:   nil,
			},
			Expected: productcatalog.ScheduledStatus,
		},
		{
			Name: "Scheduled with fixed period",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now.Add(48 * time.Hour)),
			},
			Expected: productcatalog.ScheduledStatus,
		},
		{
			Name: "Invalid with inverse period",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now.Add(24 * time.Hour)),
				EffectiveTo:   lo.ToPtr(now.Add(-24 * time.Hour)),
			},
			Expected: productcatalog.InvalidStatus,
		},
		{
			Name: "Invalid with no start with end in the past",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   lo.ToPtr(now.Add(-24 * time.Hour)),
			},
			Expected: productcatalog.ArchivedStatus,
		},
		{
			Name: "Invalid with no start with end in the future",
			Effective: productcatalog.EffectivePeriod{
				EffectiveFrom: nil,
				EffectiveTo:   lo.ToPtr(now.Add(24 * time.Hour)),
			},
			Expected: productcatalog.ActiveStatus,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, test.Effective.Status())
		})
	}
}
