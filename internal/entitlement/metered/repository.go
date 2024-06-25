package meteredentitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type UsageResetRepo interface {
	Save(ctx context.Context, usageResetTime UsageResetTime) error
	GetLastAt(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (*UsageResetTime, error)
	GetBetween(ctx context.Context, entitlementID models.NamespacedID, from time.Time, to time.Time) ([]UsageResetTime, error)

	entutils.TxCreator
	entutils.TxUser[UsageResetRepo]
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
	EntitlementID string
}
