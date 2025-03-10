package balance

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type SnapshotRepo interface {
	InvalidateAfter(ctx context.Context, owner models.NamespacedID, at time.Time) error
	// The returned Snapshot might not have usage data.
	GetLatestValidAt(ctx context.Context, owner models.NamespacedID, at time.Time) (Snapshot, error)
	Save(ctx context.Context, owner models.NamespacedID, balances []Snapshot) error
}

// No balance has been saved since start of measurement for the owner
type NoSavedBalanceForOwnerError struct {
	Owner models.NamespacedID
	Time  time.Time
}

func (e NoSavedBalanceForOwnerError) Error() string {
	return fmt.Sprintf("no saved balance for owner %s in namespace %s before %s", e.Owner.ID, e.Owner.Namespace, e.Time)
}
