package credit

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type BalanceSnapshotRepo interface {
	InvalidateAfter(ctx context.Context, owner NamespacedGrantOwner, at time.Time) error
	GetLatestValidAt(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (GrantBalanceSnapshot, error)
	Save(ctx context.Context, owner NamespacedGrantOwner, balances []GrantBalanceSnapshot) error

	entutils.TxCreator
	entutils.TxUser[BalanceSnapshotRepo]
}

type txRepo struct {
	BalanceSnapshotRepo
}

func (t *txRepo) InvalidateAfter(ctx context.Context, owner NamespacedGrantOwner, at time.Time) error {
	return t.tx(ctx).InvalidateAfter(ctx, owner, at)
}

func (t *txRepo) GetLatestValidAt(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (GrantBalanceSnapshot, error) {
	return t.tx(ctx).GetLatestValidAt(ctx, owner, at)
}

func (t *txRepo) Save(ctx context.Context, owner NamespacedGrantOwner, balances []GrantBalanceSnapshot) error {
	return t.tx(ctx).Save(ctx, owner, balances)
}

// FIXME: this should be hidden and happen in the extracted repository level
func (t *txRepo) tx(ctx context.Context) BalanceSnapshotRepo {
	if ctxTx, err := entutils.GetTxDriver(ctx); err == nil {
		// we're already in a tx
		return t.BalanceSnapshotRepo.WithTx(ctx, ctxTx)
	} else {
		return t.BalanceSnapshotRepo
	}
}

type BalanceSnapshotConnector interface {
	InvalidateAfter(ctx context.Context, owner NamespacedGrantOwner, at time.Time) error
	GetLatestValidAt(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (GrantBalanceSnapshot, error)
	Save(ctx context.Context, owner NamespacedGrantOwner, balances []GrantBalanceSnapshot) error
}

type balanceSnapshotConnector struct {
	balanceSnapshotRepo BalanceSnapshotRepo
}

func NewBalanceSnapshotConnector(
	balanceSnapshotRepo BalanceSnapshotRepo,
) BalanceSnapshotConnector {
	return &balanceSnapshotConnector{
		balanceSnapshotRepo: &txRepo{balanceSnapshotRepo},
	}
}

func (b *balanceSnapshotConnector) InvalidateAfter(ctx context.Context, owner NamespacedGrantOwner, at time.Time) error {
	return b.balanceSnapshotRepo.InvalidateAfter(ctx, owner, at)
}

func (b *balanceSnapshotConnector) GetLatestValidAt(ctx context.Context, owner NamespacedGrantOwner, at time.Time) (GrantBalanceSnapshot, error) {
	return b.balanceSnapshotRepo.GetLatestValidAt(ctx, owner, at)
}

func (b *balanceSnapshotConnector) Save(ctx context.Context, owner NamespacedGrantOwner, balances []GrantBalanceSnapshot) error {
	return b.balanceSnapshotRepo.Save(ctx, owner, balances)
}
