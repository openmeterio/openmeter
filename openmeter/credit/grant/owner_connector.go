package grant

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type EndCurrentUsagePeriodParams struct {
	At           time.Time
	RetainAnchor bool
}

type OwnerMeter struct {
	Meter         meter.Meter
	DefaultParams streaming.QueryParams
	SubjectKey    string
}

type OwnerConnector interface {
	GetMeter(ctx context.Context, owner NamespacedOwner) (*OwnerMeter, error)
	GetStartOfMeasurement(ctx context.Context, owner NamespacedOwner) (time.Time, error)
	GetPeriodStartTimesBetween(ctx context.Context, owner NamespacedOwner, from, to time.Time) ([]time.Time, error)
	GetUsagePeriodStartAt(ctx context.Context, owner NamespacedOwner, at time.Time) (time.Time, error)
	GetOwnerSubjectKey(ctx context.Context, owner NamespacedOwner) (string, error)

	EndCurrentUsagePeriod(ctx context.Context, owner NamespacedOwner, params EndCurrentUsagePeriodParams) error
	LockOwnerForTx(ctx context.Context, owner NamespacedOwner) error
}

type OwnerNotFoundError struct {
	Owner          NamespacedOwner
	AttemptedOwner string
}

func (e OwnerNotFoundError) Error() string {
	return fmt.Sprintf("Owner %s not found in namespace %s, attempted to find as %s", e.Owner.ID, e.Owner.Namespace, e.AttemptedOwner)
}
