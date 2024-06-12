package credit

import (
	"context"
	"time"
)

type GrantBalanceConnector interface {
	InvalidateAfter(ctx context.Context, owner NamespacedGrantOwner, effectiveAt time.Time) error
	GetLatestValidAt(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (GrantBalanceMap, error)
	Save(ctx context.Context, owner NamespacedGrantOwner, balances []GrantBalanceMap) error
}
