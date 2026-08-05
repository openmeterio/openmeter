package balance

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type SnapshotService interface {
	InvalidateAfter(ctx context.Context, owner models.NamespacedID, at time.Time) error
	GetLatestValidAt(ctx context.Context, owner models.NamespacedID, at time.Time) (Snapshot, error)
	GetLatestValidCompleteAt(ctx context.Context, owner models.NamespacedID, at time.Time) (Snapshot, error)
	// Save persists legacy snapshots without complete usage-period state.
	Save(ctx context.Context, owner models.NamespacedID, balances []Snapshot) error
	// SaveComplete requires and persists complete usage-period state.
	SaveComplete(ctx context.Context, owner models.NamespacedID, balances []Snapshot) error
	// To make sure repo doesn't implement the service interface
	service()
}

type SnapshotServiceConfig struct {
	Repo SnapshotRepo
}

type service struct {
	SnapshotServiceConfig
}

func NewSnapshotService(conf SnapshotServiceConfig) SnapshotService {
	return &service{
		SnapshotServiceConfig: conf,
	}
}

func (s *service) service() {}

func (s *service) InvalidateAfter(ctx context.Context, owner models.NamespacedID, at time.Time) error {
	return s.Repo.InvalidateAfter(ctx, owner, at)
}

func (s *service) GetLatestValidAt(ctx context.Context, owner models.NamespacedID, at time.Time) (Snapshot, error) {
	return s.Repo.GetLatestValidAt(ctx, owner, at)
}

func (s *service) GetLatestValidCompleteAt(ctx context.Context, owner models.NamespacedID, at time.Time) (Snapshot, error) {
	return s.Repo.GetLatestValidCompleteAt(ctx, owner, at)
}

func (s *service) Save(ctx context.Context, owner models.NamespacedID, balances []Snapshot) error {
	legacySnapshots := make([]Snapshot, len(balances))
	for i, snapshot := range balances {
		legacySnapshots[i] = snapshot.Clone()
		legacySnapshots[i].UsageSnapshot = nil
	}

	return s.Repo.Save(ctx, owner, legacySnapshots)
}

func (s *service) SaveComplete(ctx context.Context, owner models.NamespacedID, balances []Snapshot) error {
	for _, snapshot := range balances {
		if snapshot.UsageSnapshot == nil {
			return fmt.Errorf("cannot save incomplete balance snapshot at %s", snapshot.At)
		}
	}

	return s.Repo.Save(ctx, owner, balances)
}
