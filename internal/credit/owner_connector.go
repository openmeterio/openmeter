package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type EndCurrentUsagePeriodParams struct {
	At           time.Time
	RetainAnchor bool
}

type OwnerConnector interface {
	GetOwnerQueryParams(ctx context.Context, owner NamespacedGrantOwner) (meterSlug string, defaultParams *streaming.QueryParams, err error)
	GetStartOfMeasurement(ctx context.Context, owner NamespacedGrantOwner) (time.Time, error)
	GetLastReset(ctx context.Context, owner NamespacedGrantOwner) (time.Time, error)
	GetPeriodStartTimesBetween(ctx context.Context, owner NamespacedGrantOwner, from, to time.Time) ([]time.Time, error)
	GetUsagePeriodStartAt(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (time.Time, error)

	//FIXME: this is a terrible hack
	EndCurrentUsagePeriodTx(ctx context.Context, tx *entutils.TxDriver, owner NamespacedGrantOwner, params EndCurrentUsagePeriodParams) error
	//FIXME: this is a terrible hack
	LockOwnerForTx(ctx context.Context, tx *entutils.TxDriver, owner NamespacedGrantOwner) error
}

type OwnerNotFoundError struct {
	Owner          NamespacedGrantOwner
	AttemptedOwner string
}

func (e OwnerNotFoundError) Error() string {
	return fmt.Sprintf("Owner %s not found in namespace %s, attempted to find as %s", e.Owner.ID, e.Owner.Namespace, e.AttemptedOwner)
}
