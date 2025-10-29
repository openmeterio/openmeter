package balanceworker

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGetEntitlementActivityPeriod(t *testing.T) {
	tests := []struct {
		name string
		ent  ListAffectedEntitlementsResponse
		want timeutil.StartBoundedPeriod
	}{
		{
			name: "pre-active from entitlement", // Active from is null
			ent: ListAffectedEntitlementsResponse{
				CreatedAt: testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z"),
			},
			want: timeutil.StartBoundedPeriod{
				From: testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z"),
			},
		},
		{
			name: "pre-active from deleted entitlement",
			ent: ListAffectedEntitlementsResponse{
				CreatedAt: testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z"),
				DeletedAt: lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z")),
			},
			want: timeutil.StartBoundedPeriod{
				From: testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z"),
				To:   lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z")),
			},
		},
		{
			name: "entitlement with active from",
			ent: ListAffectedEntitlementsResponse{
				CreatedAt:  testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z"),
				ActiveFrom: lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z")),
				ActiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T03:00:00Z")),
			},
			want: timeutil.StartBoundedPeriod{
				From: testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z"),
				To:   lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T03:00:00Z")),
			},
		},
		{
			name: "entitlement with active from only",
			ent: ListAffectedEntitlementsResponse{
				CreatedAt:  testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z"),
				ActiveFrom: lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z")),
			},
			want: timeutil.StartBoundedPeriod{
				From: testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z"),
			},
		},
		{
			name: "deleted entitlement with active to #1",
			ent: ListAffectedEntitlementsResponse{
				CreatedAt:  testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z"),
				DeletedAt:  lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T03:00:00Z")),
				ActiveFrom: lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z")),
				ActiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T04:00:00Z")),
			},
			want: timeutil.StartBoundedPeriod{
				From: testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z"),
				To:   lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T03:00:00Z")),
			},
		},
		{
			name: "deleted entitlement with active to #2",
			ent: ListAffectedEntitlementsResponse{
				CreatedAt:  testutils.GetRFC3339Time(t, "2021-01-01T01:00:00Z"),
				DeletedAt:  lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T04:00:00Z")),
				ActiveFrom: lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z")),
				ActiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T03:00:00Z")),
			},
			want: timeutil.StartBoundedPeriod{
				From: testutils.GetRFC3339Time(t, "2021-01-01T02:00:00Z"),
				To:   lo.ToPtr(testutils.GetRFC3339Time(t, "2021-01-01T03:00:00Z")),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.ent.GetEntitlementActivityPeriod()
			assert.Equal(t, test.want, got)
		})
	}
}
