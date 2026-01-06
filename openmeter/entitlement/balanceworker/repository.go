package balanceworker

import (
	"context"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type BalanceWorkerRepository interface {
	ListEntitlementsAffectedByIngestEvents(ctx context.Context, filters IngestEventQueryFilter) ([]ListAffectedEntitlementsResponse, error)
}

type IngestEventQueryFilter struct {
	Namespace    string
	EventSubject string
	MeterSlugs   []string
}

type ListAffectedEntitlementsResponse struct {
	Namespace     string
	EntitlementID string
	CreatedAt     time.Time
	DeletedAt     *time.Time
	ActiveFrom    *time.Time
	ActiveTo      *time.Time
}

// GetEntitlementActivityPeriod returns the period where the entitlement could have received events.
func (r *ListAffectedEntitlementsResponse) GetEntitlementActivityPeriod() timeutil.StartBoundedPeriod {
	validityCandidates := lo.Filter([]*time.Time{r.ActiveTo, r.DeletedAt}, func(t *time.Time, _ int) bool {
		return t != nil
	})

	var validityEnd *time.Time
	if len(validityCandidates) > 0 {
		validityEnd = lo.MinBy(validityCandidates, func(a, b *time.Time) bool {
			return a.Before(*b)
		})
	}

	if r.ActiveFrom != nil {
		return timeutil.StartBoundedPeriod{
			From: *r.ActiveFrom,
			To:   validityEnd,
		}
	}

	return timeutil.StartBoundedPeriod{
		From: r.CreatedAt,
		To:   validityEnd,
	}
}
