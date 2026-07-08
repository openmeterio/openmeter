package clickhouse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPlanCachePopulation(t *testing.T) {
	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	now := base.Add(30 * 24 * time.Hour)
	hour := func(h int) time.Time { return base.Add(time.Duration(h) * time.Hour) }

	freshClaim := func(from, until time.Time) *cacheCoverage {
		return &cacheCoverage{From: from, Until: until, FirstWrittenAt: now.Add(-time.Hour), PopulatedAt: now.Add(-time.Hour)}
	}

	tests := []struct {
		name          string
		lo, hi        time.Time
		cov           *cacheCoverage
		invalidatedAt time.Time
		wantPopulate  []timeRange
		wantStore     *cacheCoverage
	}{
		{
			name: "no claim populates everything and starts a claim",
			lo:   hour(0), hi: hour(10),
			wantPopulate: []timeRange{{From: hour(0), To: hour(10)}},
			wantStore:    &cacheCoverage{From: hour(0), Until: hour(10), FirstWrittenAt: now, PopulatedAt: now},
		},
		{
			name: "fully covered populates and stores nothing",
			lo:   hour(2), hi: hour(8),
			cov: freshClaim(hour(0), hour(10)),
		},
		{
			name: "exactly covered populates and stores nothing",
			lo:   hour(0), hi: hour(10),
			cov: freshClaim(hour(0), hour(10)),
		},
		{
			name: "missing prefix populates only the prefix and extends the claim",
			lo:   hour(0), hi: hour(8),
			cov:          freshClaim(hour(4), hour(10)),
			wantPopulate: []timeRange{{From: hour(0), To: hour(4)}},
			wantStore:    &cacheCoverage{From: hour(0), Until: hour(10), FirstWrittenAt: now.Add(-time.Hour), PopulatedAt: now},
		},
		{
			name: "missing suffix populates only the suffix and extends the claim",
			lo:   hour(2), hi: hour(15),
			cov:          freshClaim(hour(0), hour(10)),
			wantPopulate: []timeRange{{From: hour(10), To: hour(15)}},
			wantStore:    &cacheCoverage{From: hour(0), Until: hour(15), FirstWrittenAt: now.Add(-time.Hour), PopulatedAt: now},
		},
		{
			name: "missing both sides populates prefix and suffix and extends both ways",
			lo:   hour(0), hi: hour(15),
			cov:          freshClaim(hour(4), hour(10)),
			wantPopulate: []timeRange{{From: hour(0), To: hour(4)}, {From: hour(10), To: hour(15)}},
			wantStore:    &cacheCoverage{From: hour(0), Until: hour(15), FirstWrittenAt: now.Add(-time.Hour), PopulatedAt: now},
		},
		{
			name: "adjacent below is an extension, not disjoint",
			lo:   hour(0), hi: hour(4),
			cov:          freshClaim(hour(4), hour(10)),
			wantPopulate: []timeRange{{From: hour(0), To: hour(4)}},
			wantStore:    &cacheCoverage{From: hour(0), Until: hour(10), FirstWrittenAt: now.Add(-time.Hour), PopulatedAt: now},
		},
		{
			name: "disjoint below populates everything but keeps the stored claim",
			lo:   hour(0), hi: hour(3),
			cov:          freshClaim(hour(4), hour(10)),
			wantPopulate: []timeRange{{From: hour(0), To: hour(3)}},
		},
		{
			name: "disjoint above populates everything but keeps the stored claim",
			lo:   hour(12), hi: hour(15),
			cov:          freshClaim(hour(4), hour(10)),
			wantPopulate: []timeRange{{From: hour(12), To: hour(15)}},
		},
		{
			name: "expired claim is ignored: full populate and a fresh claim",
			lo:   hour(0), hi: hour(10),
			cov:          &cacheCoverage{From: hour(0), Until: hour(10), FirstWrittenAt: now.Add(-cacheCoverageTrustWindow - time.Minute), PopulatedAt: now.Add(-time.Hour)},
			wantPopulate: []timeRange{{From: hour(0), To: hour(10)}},
			wantStore:    &cacheCoverage{From: hour(0), Until: hour(10), FirstWrittenAt: now, PopulatedAt: now},
		},
		{
			name: "claim at exactly the trust window boundary is still honored",
			lo:   hour(0), hi: hour(10),
			cov: &cacheCoverage{From: hour(0), Until: hour(10), FirstWrittenAt: now.Add(-cacheCoverageTrustWindow), PopulatedAt: now.Add(-time.Hour)},
		},
		{
			// The blocker class: a claim planned BEFORE an invalidation marker
			// may describe rows the invalidation wiped — it must be ignored even
			// though its row landed (created_at) after the marker.
			name: "claim planned before the invalidation marker is distrusted",
			lo:   hour(0), hi: hour(10),
			cov:           freshClaim(hour(0), hour(10)), // PopulatedAt = now-1h
			invalidatedAt: now.Add(-30 * time.Minute),    // marker after the plan
			wantPopulate:  []timeRange{{From: hour(0), To: hour(10)}},
			wantStore:     &cacheCoverage{From: hour(0), Until: hour(10), FirstWrittenAt: now, PopulatedAt: now},
		},
		{
			name: "claim planned safely after the marker is honored",
			lo:   hour(0), hi: hour(10),
			cov:           freshClaim(hour(0), hour(10)),                                      // PopulatedAt = now-1h
			invalidatedAt: now.Add(-time.Hour - cacheCoverageClockSkewMargin - 2*time.Second), // marker clearly before the plan
		},
		{
			// Within the skew margin the ordering is ambiguous (app clock vs
			// ClickHouse clock) — distrust, at the cost of one redundant populate.
			name: "claim inside the clock-skew margin of the marker is distrusted",
			lo:   hour(0), hi: hour(10),
			cov:           freshClaim(hour(0), hour(10)),                        // PopulatedAt = now-1h
			invalidatedAt: now.Add(-time.Hour - cacheCoverageClockSkewMargin/2), // marker just before the plan
			wantPopulate:  []timeRange{{From: hour(0), To: hour(10)}},
			wantStore:     &cacheCoverage{From: hour(0), Until: hour(10), FirstWrittenAt: now, PopulatedAt: now},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := planCachePopulation(tt.lo, tt.hi, tt.cov, tt.invalidatedAt, now)
			assert.Equal(t, tt.wantPopulate, plan.Populate)
			assert.Equal(t, tt.wantStore, plan.Store)
		})
	}
}
