package balance

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type SnapshotService interface {
	InvalidateAfter(ctx context.Context, owner models.NamespacedID, at time.Time) error
	GetLatestValidAt(ctx context.Context, owner models.NamespacedID, at time.Time) (Snapshot, error)
	Save(ctx context.Context, owner models.NamespacedID, balances []Snapshot) error
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

func (s *service) Save(ctx context.Context, owner models.NamespacedID, balances []Snapshot) error {
	return s.Repo.Save(ctx, owner, balances)
}
