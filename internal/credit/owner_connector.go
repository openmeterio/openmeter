package credit

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/internal/streaming"
)

type OwnerConnector interface {
	GetOwnerQueryParams(ctx context.Context, owner NamespacedGrantOwner) (namespace string, defaultParams streaming.QueryParams, err error)
	GetStartOfMeasurement(ctx context.Context, owner NamespacedGrantOwner) (time.Time, error)
	GetPeriodStartTimesBetween(ctx context.Context, owner NamespacedGrantOwner, from, to time.Time) ([]time.Time, error)
	GetCurrentUsagePeriodStart(ctx context.Context, owner NamespacedGrantOwner) (time.Time, error)
	EndCurrentUsagePeriod(ctx context.Context, owner NamespacedGrantOwner, at time.Time) error
}
