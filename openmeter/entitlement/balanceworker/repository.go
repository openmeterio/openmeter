package balanceworker

import (
	"context"
	"errors"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
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

func (f IngestEventQueryFilter) Validate() error {
	var errs []error

	if f.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if f.EventSubject == "" {
		errs = append(errs, errors.New("subject is required"))
	}

	if len(f.MeterSlugs) == 0 {
		errs = append(errs, errors.New("at least one meter key is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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
