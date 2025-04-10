package balance_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// MockSnapshotRepo is a mock implementation of balance.SnapshotRepo
type MockSnapshotRepo struct {
	snapshots map[string]balance.Snapshot
}

func NewMockSnapshotRepo() *MockSnapshotRepo {
	return &MockSnapshotRepo{
		snapshots: make(map[string]balance.Snapshot),
	}
}

func (m *MockSnapshotRepo) InvalidateAfter(ctx context.Context, owner models.NamespacedID, at time.Time) error {
	return nil
}

func (m *MockSnapshotRepo) GetLatestValidAt(ctx context.Context, owner models.NamespacedID, at time.Time) (balance.Snapshot, error) {
	key := owner.Namespace + ":" + owner.ID
	snapshot, ok := m.snapshots[key]
	if !ok {
		return balance.Snapshot{}, balance.NoSavedBalanceForOwnerError{
			Owner: owner,
			Time:  at,
		}
	}
	return snapshot, nil
}

func (m *MockSnapshotRepo) Save(ctx context.Context, owner models.NamespacedID, balances []balance.Snapshot) error {
	key := owner.Namespace + ":" + owner.ID
	if len(balances) > 0 {
		m.snapshots[key] = balances[0]
	}
	return nil
}

// MockOwnerConnector is a mock implementation of grant.OwnerConnector
type MockOwnerConnector struct {
	usagePeriodStartAt time.Time
	meterSlug          string
}

func NewMockOwnerConnector(usagePeriodStartAt time.Time, meterSlug string) *MockOwnerConnector {
	return &MockOwnerConnector{
		usagePeriodStartAt: usagePeriodStartAt,
		meterSlug:          meterSlug,
	}
}

func (m *MockOwnerConnector) DescribeOwner(ctx context.Context, id models.NamespacedID) (grant.Owner, error) {
	return grant.Owner{
		NamespacedID: id,
		Meter: meter.Meter{
			Key:         m.meterSlug,
			Aggregation: meter.MeterAggregationSum,
		},
		DefaultQueryParams: streaming.QueryParams{
			FilterSubject: []string{"subject1"},
		},
	}, nil
}

func (m *MockOwnerConnector) GetResetTimelineInclusive(ctx context.Context, id models.NamespacedID, period timeutil.ClosedPeriod) (timeutil.SimpleTimeline, error) {
	return timeutil.SimpleTimeline{}, nil
}

func (m *MockOwnerConnector) GetUsagePeriodStartAt(ctx context.Context, id models.NamespacedID, at time.Time) (time.Time, error) {
	return m.usagePeriodStartAt, nil
}

func (m *MockOwnerConnector) GetStartOfMeasurement(ctx context.Context, id models.NamespacedID) (time.Time, error) {
	return m.usagePeriodStartAt, nil
}

func (m *MockOwnerConnector) EndCurrentUsagePeriod(ctx context.Context, id models.NamespacedID, params grant.EndCurrentUsagePeriodParams) error {
	return nil
}

func (m *MockOwnerConnector) LockOwnerForTx(ctx context.Context, id models.NamespacedID) error {
	return nil
}

func TestGetLatestValidAt(t *testing.T) {
	// Common setup
	ctx := context.Background()
	now := time.Now().UTC()
	periodStart := now.Add(-24 * time.Hour)
	meterSlug := "test-meter"

	// Create owner
	owner := models.NamespacedID{
		Namespace: "test-namespace",
		ID:        "test-owner",
	}

	t.Run("Should fill usage if snapshot has zero usage", func(t *testing.T) {
		// Create mock streaming connector
		streamingConnector := testutils.NewMockStreamingConnector(t)

		// Add usage data to the mock streaming connector
		streamingConnector.AddSimpleEvent(meterSlug, 100.0, now.Add(-12*time.Hour))

		// Create mock snapshot repo with a snapshot that has zero usage
		mockRepo := NewMockSnapshotRepo()
		mockRepo.snapshots[owner.Namespace+":"+owner.ID] = balance.Snapshot{
			Usage:    balance.SnapshottedUsage{}, // Zero usage
			Balances: balance.Map{"grant1": 1000.0},
			At:       now.Add(-1 * time.Hour),
		}

		// Create mock owner connector
		mockOwnerConnector := NewMockOwnerConnector(periodStart, meterSlug)

		// Create the service
		service := balance.NewSnapshotService(balance.SnapshotServiceConfig{
			OwnerConnector:     mockOwnerConnector,
			StreamingConnector: streamingConnector,
			Repo:               mockRepo,
		})

		// Test
		snapshot, err := service.GetLatestValidAt(ctx, owner, now)

		// Verify
		require.NoError(t, err)
		assert.Equal(t, 100.0, snapshot.Usage.Usage, "Usage should be filled with the value from streaming connector")
		assert.Equal(t, periodStart, snapshot.Usage.Since, "Usage since should be set to the period start")
		assert.Equal(t, 1000.0, snapshot.Balances["grant1"], "Balance should remain unchanged")
	})

	t.Run("Should preserve existing usage if snapshot already has usage data", func(t *testing.T) {
		// Create mock streaming connector
		streamingConnector := testutils.NewMockStreamingConnector(t)

		// Add usage data to the mock streaming connector
		streamingConnector.AddSimpleEvent(meterSlug, 100.0, now.Add(-12*time.Hour))

		// Create mock snapshot repo with a snapshot that already has usage data
		mockRepo := NewMockSnapshotRepo()
		mockRepo.snapshots[owner.Namespace+":"+owner.ID] = balance.Snapshot{
			Usage: balance.SnapshottedUsage{
				Usage: 50.0,
				Since: periodStart,
			},
			Balances: balance.Map{"grant1": 1000.0},
			At:       now.Add(-1 * time.Hour),
		}

		// Create mock owner connector
		mockOwnerConnector := NewMockOwnerConnector(periodStart, meterSlug)

		// Create the service
		service := balance.NewSnapshotService(balance.SnapshotServiceConfig{
			OwnerConnector:     mockOwnerConnector,
			StreamingConnector: streamingConnector,
			Repo:               mockRepo,
		})

		// Test
		snapshot, err := service.GetLatestValidAt(ctx, owner, now)

		// Verify
		require.NoError(t, err)
		assert.Equal(t, 50.0, snapshot.Usage.Usage, "Usage should remain unchanged")
		assert.Equal(t, periodStart, snapshot.Usage.Since, "Usage since should remain unchanged")
		assert.Equal(t, 1000.0, snapshot.Balances["grant1"], "Balance should remain unchanged")
	})

	t.Run("Should return error if no snapshot exists", func(t *testing.T) {
		// Create mock streaming connector
		streamingConnector := testutils.NewMockStreamingConnector(t)

		// Add usage data to the mock streaming connector
		streamingConnector.AddSimpleEvent(meterSlug, 100.0, now.Add(-12*time.Hour))

		// Create empty mock snapshot repo (no snapshots)
		mockRepo := NewMockSnapshotRepo()

		// Create mock owner connector
		mockOwnerConnector := NewMockOwnerConnector(periodStart, meterSlug)

		// Create the service
		service := balance.NewSnapshotService(balance.SnapshotServiceConfig{
			OwnerConnector:     mockOwnerConnector,
			StreamingConnector: streamingConnector,
			Repo:               mockRepo,
		})

		// Test
		_, err := service.GetLatestValidAt(ctx, owner, now)

		// Verify
		require.Error(t, err)
		_, isNoSavedBalanceErr := err.(balance.NoSavedBalanceForOwnerError)
		assert.True(t, isNoSavedBalanceErr, "Expected NoSavedBalanceForOwnerError")
	})
}
