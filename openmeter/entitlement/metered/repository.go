package meteredentitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type UsageResetRepo interface {
	Save(ctx context.Context, usageResetTime UsageResetTime) error
	GetLastAt(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (UsageResetTime, error)
	GetBetween(ctx context.Context, entitlementID models.NamespacedID, period timeutil.ClosedPeriod) ([]UsageResetTime, error)
}

type UsageResetNotFoundError struct {
	EntitlementID models.NamespacedID
}

func (e UsageResetNotFoundError) Error() string {
	return fmt.Sprintf("usage reset not found for entitlement %s in namespace %s", e.EntitlementID.ID, e.EntitlementID.Namespace)
}

type UsageResetTime struct {
	models.NamespacedModel
	ResetTime     time.Time
	Anchor        time.Time
	EntitlementID string
}
