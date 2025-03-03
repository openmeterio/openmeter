package grant

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type EndCurrentUsagePeriodParams struct {
	At           time.Time
	RetainAnchor bool
}

type OwnerMeter struct {
	Meter         meter.Meter
	DefaultParams streaming.QueryParams
}

type ResetBehavior struct {
	PreserveOverage bool
}

type OwnerConnector interface {
	GetMeter(ctx context.Context, id models.NamespacedID) (*OwnerMeter, error)
	GetStartOfMeasurement(ctx context.Context, id models.NamespacedID) (time.Time, error)
	// Returns all manual and programmatic reset times effective for any time in the period (start and end inclusive)
	// "reset times effective" means that:
	// let LR(t Time) be the last reset time before t
	// GetResetTimelineInclusive(period) = for t in [period.From, period.To]: LR(t)
	// This means, the first time can be before the input period (except if the start of the period is a reset itself)
	GetResetTimelineInclusive(ctx context.Context, id models.NamespacedID, period timeutil.Period) (timeutil.SimpleTimeline, error)
	GetResetBehavior(ctx context.Context, id models.NamespacedID) (ResetBehavior, error)
	GetUsagePeriodStartAt(ctx context.Context, id models.NamespacedID, at time.Time) (time.Time, error)
	GetOwnerSubjectKey(ctx context.Context, id models.NamespacedID) (string, error)

	EndCurrentUsagePeriod(ctx context.Context, id models.NamespacedID, params EndCurrentUsagePeriodParams) error
	LockOwnerForTx(ctx context.Context, id models.NamespacedID) error
}

type OwnerNotFoundError struct {
	Owner          models.NamespacedID
	AttemptedOwner string
}

func (e OwnerNotFoundError) Error() string {
	return fmt.Sprintf("Owner %s not found in namespace %s, attempted to find as %s", e.Owner.ID, e.Owner.Namespace, e.AttemptedOwner)
}
