package balance

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type SnapshotRepo interface {
	InvalidateAfter(ctx context.Context, owner grant.NamespacedOwner, at time.Time) error
	GetLatestValidAt(ctx context.Context, owner grant.NamespacedOwner, at time.Time) (Snapshot, error)
	Save(ctx context.Context, owner grant.NamespacedOwner, balances []Snapshot) error

	entutils.TxCreator
	entutils.TxUser[SnapshotRepo]
}

// No balance has been saved since start of measurement for the owner
type NoSavedBalanceForOwnerError struct {
	Owner grant.NamespacedOwner
	Time  time.Time
}

func (e NoSavedBalanceForOwnerError) Error() string {
	return fmt.Sprintf("no saved balance for owner %s in namespace %s before %s", e.Owner.ID, e.Owner.Namespace, e.Time)
}
