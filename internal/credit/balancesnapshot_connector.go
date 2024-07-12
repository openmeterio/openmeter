package credit

import (
	"context"
	"fmt"
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
	Save(ctx context.Context, owner NamespacedGrantOwner, grantMap map[string]Grant, snap GrantBalanceSnapshot) error
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

func (b *balanceSnapshotConnector) Save(ctx context.Context, owner NamespacedGrantOwner, grantMap map[string]Grant, snap GrantBalanceSnapshot) error {
	filtered, err := b.filterBalances(snap.Balances, grantMap, snap.At)
	if err != nil {
		return err
	}
	snap.Balances = *filtered
	return b.balanceSnapshotRepo.Save(ctx, owner, []GrantBalanceSnapshot{snap})
}

// filterBalances filters out balances for grants that are not active at the given time
func (b *balanceSnapshotConnector) filterBalances(balances GrantBalanceMap, grants map[string]Grant, at time.Time) (*GrantBalanceMap, error) {
	filtered := &GrantBalanceMap{}
	for grantID, grantBalance := range balances {
		grant, ok := grants[grantID]
		// inconsistency check, shouldn't happen
		if !ok {
			return nil, fmt.Errorf("received balance for unknown grant: %s", grantID)
		}

		if !grant.ActiveAt(at) {
			continue
		}

		filtered.Set(grantID, grantBalance)
	}
	return filtered, nil
}
