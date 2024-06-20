package credit

import (
	"context"
	"fmt"
	"time"
)

type GrantBalanceConnector interface {
	InvalidateAfter(ctx context.Context, owner NamespacedGrantOwner, at time.Time) error
	GetLatestValidAt(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (GrantBalanceSnapshot, error)
	Save(ctx context.Context, owner NamespacedGrantOwner, balances []GrantBalanceSnapshot) error
}

// No balance has been saved since start of measurement for the owner
type GrantBalanceNoSavedBalanceForOwnerError struct {
	Owner NamespacedGrantOwner
	Time  time.Time
}

func (e GrantBalanceNoSavedBalanceForOwnerError) Error() string {
	return fmt.Sprintf("no saved balance for owner %s in namespace %s before %s", e.Owner.ID, e.Owner.Namespace, e.Time)
}
