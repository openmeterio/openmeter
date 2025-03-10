package balance

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type SnapshotService interface {
	InvalidateAfter(ctx context.Context, owner models.NamespacedID, at time.Time) error
	GetLatestValidAt(ctx context.Context, owner models.NamespacedID, at time.Time) (Snapshot, error)
	Save(ctx context.Context, owner models.NamespacedID, balances []Snapshot) error
	// To make sure repo doesn't implement the service interface
	service()
}

type SnapshotServiceConfig struct {
	OwnerConnector     grant.OwnerConnector
	StreamingConnector streaming.Connector
	Repo               SnapshotRepo
}

type service struct {
	UsageQuerier UsageQuerier
	SnapshotServiceConfig
}

func NewSnapshotService(conf SnapshotServiceConfig) SnapshotService {
	return &service{
		SnapshotServiceConfig: conf,
		// We build a custom UsageQuerier for our usecase here
		UsageQuerier: NewUsageQuerier(UsageQuerierConfig{
			StreamingConnector: conf.StreamingConnector,
			DescribeOwner:      conf.OwnerConnector.DescribeOwner,
			GetDefaultParams: func(ctx context.Context, ownerID models.NamespacedID) (streaming.QueryParams, error) {
				owner, err := conf.OwnerConnector.DescribeOwner(ctx, ownerID)
				if err != nil {
					return streaming.QueryParams{}, err
				}
				return owner.DefaultQueryParams, nil
			},
			GetUsagePeriodStartAt: conf.OwnerConnector.GetUsagePeriodStartAt,
		}),
	}
}

func (s *service) service() {}

func (s *service) InvalidateAfter(ctx context.Context, owner models.NamespacedID, at time.Time) error {
	return s.Repo.InvalidateAfter(ctx, owner, at)
}

func (s *service) GetLatestValidAt(ctx context.Context, owner models.NamespacedID, at time.Time) (Snapshot, error) {
	res, err := s.Repo.GetLatestValidAt(ctx, owner, at)
	if err != nil {
		return Snapshot{}, err
	}

	// We have to manually fill in the usage data if it wasn't saved
	if res.Usage.IsZero() {
		periodStart, err := s.OwnerConnector.GetUsagePeriodStartAt(ctx, owner, res.At)
		if err != nil {
			return Snapshot{}, err
		}

		usage, err := s.UsageQuerier.QueryUsage(ctx, owner, timeutil.Period{
			From: periodStart,
			To:   res.At,
		})
		if err != nil {
			return Snapshot{}, err
		}

		res.Usage = SnapshottedUsage{
			Usage: usage,
			Since: periodStart,
		}
	}

	return res, nil
}

func (s *service) Save(ctx context.Context, owner models.NamespacedID, balances []Snapshot) error {
	return s.Repo.Save(ctx, owner, balances)
}
