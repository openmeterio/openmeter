package balance

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type SnapshotInvalidationVersion uint64

type SnapshotRepo interface {
	InvalidateAfter(ctx context.Context, owner models.NamespacedID, at time.Time) error
	GetInvalidationVersion(ctx context.Context, owner models.NamespacedID) (SnapshotInvalidationVersion, error)
	// GetLatestValidAt returns the latest complete snapshot.
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
