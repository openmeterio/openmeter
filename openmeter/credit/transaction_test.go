package credit

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type testTransactable struct {
	committed bool
}

func (d *testTransactable) Commit() error {
	d.committed = true
	return nil
}

func (d *testTransactable) Rollback() error         { return nil }
func (d *testTransactable) SavePoint(string) error  { return nil }
func (d *testTransactable) RollbackTo(string) error { return nil }
func (d *testTransactable) Release(string) error    { return nil }

func newTestTransaction() (*entutils.TxDriver, *testTransactable) {
	driver := &testTransactable{}
	return entutils.NewTxDriver(driver, &entutils.RawEntConfig{}), driver
}

type testGrantRepo struct {
	driver      transaction.Driver
	createGrant func(context.Context, grant.RepoCreateInput) (*grant.Grant, error)
	getGrant    func(context.Context, models.NamespacedID) (grant.Grant, error)
	voidGrant   func(context.Context, models.NamespacedID, time.Time) error
}

func (r *testGrantRepo) CreateGrant(ctx context.Context, input grant.RepoCreateInput) (*grant.Grant, error) {
	return r.createGrant(ctx, input)
}

func (r *testGrantRepo) VoidGrant(ctx context.Context, id models.NamespacedID, at time.Time) error {
	return r.voidGrant(ctx, id, at)
}

func (r *testGrantRepo) GetGrant(ctx context.Context, id models.NamespacedID) (grant.Grant, error) {
	return r.getGrant(ctx, id)
}

func (r *testGrantRepo) ListGrants(context.Context, grant.ListParams) (pagination.Result[grant.Grant], error) {
	return pagination.Result[grant.Grant]{}, nil
}

func (r *testGrantRepo) ListActiveGrantsBetween(context.Context, models.NamespacedID, time.Time, time.Time) ([]grant.Grant, error) {
	return nil, nil
}

func (r *testGrantRepo) DeleteOwnerGrants(context.Context, models.NamespacedID) error { return nil }

func (r *testGrantRepo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	return ctx, r.driver, nil
}

func (r *testGrantRepo) WithTx(context.Context, *entutils.TxDriver) grant.Repo { return r }
func (r *testGrantRepo) Self() grant.Repo                                      { return r }

type testOwnerConnector struct {
	describeOwner         func(context.Context, models.NamespacedID) (grant.Owner, error)
	getUsagePeriodStartAt func(context.Context, models.NamespacedID, time.Time) (time.Time, error)
	lockOwnerForTx        func(context.Context, models.NamespacedID, bool) error
}

func (c testOwnerConnector) DescribeOwner(ctx context.Context, id models.NamespacedID) (grant.Owner, error) {
	return c.describeOwner(ctx, id)
}

func (testOwnerConnector) GetResetTimelineInclusive(context.Context, models.NamespacedID, timeutil.ClosedPeriod) (timeutil.SimpleTimeline, error) {
	return timeutil.SimpleTimeline{}, nil
}

func (c testOwnerConnector) GetUsagePeriodStartAt(ctx context.Context, id models.NamespacedID, at time.Time) (time.Time, error) {
	return c.getUsagePeriodStartAt(ctx, id, at)
}

func (testOwnerConnector) GetStartOfMeasurement(context.Context, models.NamespacedID) (time.Time, error) {
	return time.Time{}, nil
}

func (testOwnerConnector) EndCurrentUsagePeriod(context.Context, models.NamespacedID, grant.EndCurrentUsagePeriodParams) error {
	return nil
}

func (c testOwnerConnector) LockOwnerForTx(ctx context.Context, id models.NamespacedID, wait bool) error {
	return c.lockOwnerForTx(ctx, id, wait)
}

type testSnapshotRepo struct {
	invalidateAfter        func(context.Context, models.NamespacedID, time.Time) error
	getInvalidationVersion func(context.Context, models.NamespacedID) (balance.SnapshotInvalidationVersion, error)
	save                   func(context.Context, models.NamespacedID, []balance.Snapshot) error
}

func (r testSnapshotRepo) InvalidateAfter(ctx context.Context, owner models.NamespacedID, at time.Time) error {
	return r.invalidateAfter(ctx, owner, at)
}

func (r testSnapshotRepo) GetInvalidationVersion(ctx context.Context, owner models.NamespacedID) (balance.SnapshotInvalidationVersion, error) {
	if r.getInvalidationVersion == nil {
		return 0, nil
	}

	return r.getInvalidationVersion(ctx, owner)
}

func (testSnapshotRepo) GetLatestValidAt(context.Context, models.NamespacedID, time.Time) (balance.Snapshot, error) {
	return balance.Snapshot{}, errors.New("not implemented")
}

func (r testSnapshotRepo) Save(ctx context.Context, owner models.NamespacedID, snapshots []balance.Snapshot) error {
	return r.save(ctx, owner, snapshots)
}

type testPublisher struct {
	eventbus.Publisher
}

func (testPublisher) Publish(context.Context, marshaler.Event) error { return nil }

type testCustomer struct {
	id string
}

func (c testCustomer) GetUsageAttribution() streaming.CustomerUsageAttribution {
	return streaming.CustomerUsageAttribution{ID: c.id}
}

func newTransactionTestConnector(t *testing.T, grantRepo grant.Repo, ownerConnector grant.OwnerConnector, snapshotRepo balance.SnapshotRepo) *connector {
	t.Helper()

	return &connector{CreditConnectorConfig: CreditConnectorConfig{
		GrantRepo:              grantRepo,
		BalanceSnapshotService: balance.NewSnapshotService(balance.SnapshotServiceConfig{Repo: snapshotRepo}),
		OwnerConnector:         ownerConnector,
		Logger:                 slog.New(slog.DiscardHandler),
		Tracer:                 noop.NewTracerProvider().Tracer("test"),
		Publisher:              testPublisher{},
		Granularity:            time.Minute,
	}}
}

func TestCreateGrantLocksOwnerBeforeReadingUsagePeriod(t *testing.T) {
	// given: usage-period state that may only be read after the owner is locked
	ownerID := models.NamespacedID{Namespace: "ns", ID: "owner"}
	effectiveAt := time.Now().UTC().Truncate(time.Minute)
	locked := false
	tx, _ := newTestTransaction()
	repo := &testGrantRepo{driver: tx}
	repo.createGrant = func(_ context.Context, input grant.RepoCreateInput) (*grant.Grant, error) {
		return &grant.Grant{
			NamespacedModel: models.NamespacedModel{Namespace: input.Namespace},
			ID:              "grant",
			OwnerID:         input.OwnerID,
			Amount:          input.Amount,
			EffectiveAt:     input.EffectiveAt,
		}, nil
	}
	repo.getGrant = func(context.Context, models.NamespacedID) (grant.Grant, error) {
		return grant.Grant{}, errors.New("unexpected grant read")
	}
	repo.voidGrant = func(context.Context, models.NamespacedID, time.Time) error {
		return errors.New("unexpected grant void")
	}

	ownerConnector := testOwnerConnector{
		describeOwner: func(context.Context, models.NamespacedID) (grant.Owner, error) {
			return grant.Owner{NamespacedID: ownerID, StreamingCustomer: testCustomer{id: "customer"}}, nil
		},
		getUsagePeriodStartAt: func(context.Context, models.NamespacedID, time.Time) (time.Time, error) {
			if !locked {
				return time.Time{}, errors.New("usage period read before owner lock")
			}
			return effectiveAt.Add(-time.Hour), nil
		},
		lockOwnerForTx: func(context.Context, models.NamespacedID, bool) error {
			locked = true
			return nil
		},
	}
	snapshotRepo := testSnapshotRepo{
		invalidateAfter: func(context.Context, models.NamespacedID, time.Time) error { return nil },
		save:            func(context.Context, models.NamespacedID, []balance.Snapshot) error { return nil },
	}
	connector := newTransactionTestConnector(t, repo, ownerConnector, snapshotRepo)

	// when: a grant is created
	_, err := connector.CreateGrant(t.Context(), ownerID, CreateGrantInput{Amount: 1, EffectiveAt: effectiveAt})

	// then: period validation observes state protected by the owner lock
	require.NoError(t, err)
	require.True(t, locked)
}

func TestVoidGrantRevalidatesAfterOwnerLock(t *testing.T) {
	// given: another transaction voids the grant between owner resolution and lock acquisition
	grantID := models.NamespacedID{Namespace: "ns", ID: "grant"}
	ownerID := models.NamespacedID{Namespace: grantID.Namespace, ID: "owner"}
	now := time.Now().UTC().Truncate(time.Minute)
	concurrentVoidAt := now.Add(-time.Minute)
	locked := false
	getCalls := 0
	voidCalls := 0
	tx, _ := newTestTransaction()
	repo := &testGrantRepo{driver: tx}
	repo.createGrant = func(context.Context, grant.RepoCreateInput) (*grant.Grant, error) {
		return nil, errors.New("unexpected grant create")
	}
	repo.getGrant = func(context.Context, models.NamespacedID) (grant.Grant, error) {
		getCalls++
		result := grant.Grant{
			NamespacedModel: models.NamespacedModel{Namespace: grantID.Namespace},
			ID:              grantID.ID,
			OwnerID:         ownerID.ID,
			EffectiveAt:     now.Add(-time.Hour),
		}
		if getCalls > 1 {
			require.True(t, locked)
			result.VoidedAt = &concurrentVoidAt
		}
		return result, nil
	}
	repo.voidGrant = func(context.Context, models.NamespacedID, time.Time) error {
		voidCalls++
		return nil
	}

	ownerConnector := testOwnerConnector{
		describeOwner: func(context.Context, models.NamespacedID) (grant.Owner, error) {
			return grant.Owner{NamespacedID: ownerID, StreamingCustomer: testCustomer{id: "customer"}}, nil
		},
		getUsagePeriodStartAt: func(context.Context, models.NamespacedID, time.Time) (time.Time, error) {
			return now.Add(-time.Hour), nil
		},
		lockOwnerForTx: func(context.Context, models.NamespacedID, bool) error {
			locked = true
			return nil
		},
	}
	snapshotRepo := testSnapshotRepo{
		invalidateAfter: func(context.Context, models.NamespacedID, time.Time) error { return nil },
		save:            func(context.Context, models.NamespacedID, []balance.Snapshot) error { return nil },
	}
	connector := newTransactionTestConnector(t, repo, ownerConnector, snapshotRepo)

	// when: the original void request acquires the owner lock
	err := connector.VoidGrant(t.Context(), grantID, &now)

	// then: it re-reads the grant and rejects the already committed void
	require.EqualError(t, err, "validation error: grant already voided")
	require.Equal(t, 2, getCalls)
	require.Zero(t, voidCalls)
}

func TestSnapshotSaveUsesOwnerLockTransaction(t *testing.T) {
	// given: an engine result with an eligible snapshot
	start := time.Now().UTC().Truncate(time.Minute)
	startingSnapshot := balance.Snapshot{
		At:            start,
		Balances:      balance.Map{},
		UsageSnapshot: &balance.UsageSnapshot{},
	}
	history, err := engine.NewGrantBurnDownHistory([]engine.GrantBurnDownHistorySegment{
		{
			ClosedPeriod:   timeutil.ClosedPeriod{From: start, To: start.Add(time.Minute)},
			BalanceAtStart: balance.Map{},
		},
		{
			ClosedPeriod:   timeutil.ClosedPeriod{From: start.Add(time.Minute), To: start.Add(2 * time.Minute)},
			BalanceAtStart: balance.Map{},
		},
	}, startingSnapshot)
	require.NoError(t, err)

	tx, driver := newTestTransaction()
	locked := false
	repo := &testGrantRepo{driver: tx}
	repo.createGrant = func(context.Context, grant.RepoCreateInput) (*grant.Grant, error) {
		return nil, errors.New("unexpected grant create")
	}
	repo.getGrant = func(context.Context, models.NamespacedID) (grant.Grant, error) {
		return grant.Grant{}, errors.New("unexpected grant read")
	}
	repo.voidGrant = func(context.Context, models.NamespacedID, time.Time) error {
		return errors.New("unexpected grant void")
	}
	ownerConnector := testOwnerConnector{
		describeOwner: func(context.Context, models.NamespacedID) (grant.Owner, error) {
			return grant.Owner{}, nil
		},
		getUsagePeriodStartAt: func(context.Context, models.NamespacedID, time.Time) (time.Time, error) {
			return time.Time{}, nil
		},
		lockOwnerForTx: func(context.Context, models.NamespacedID, bool) error {
			locked = true
			return nil
		},
	}
	snapshotRepo := testSnapshotRepo{
		invalidateAfter: func(context.Context, models.NamespacedID, time.Time) error { return nil },
		getInvalidationVersion: func(context.Context, models.NamespacedID) (balance.SnapshotInvalidationVersion, error) {
			return 4, nil
		},
		save: func(ctx context.Context, _ models.NamespacedID, _ []balance.Snapshot) error {
			require.True(t, locked)
			require.False(t, driver.committed)
			_, err := transaction.GetDriverFromContext(ctx)
			return err
		},
	}
	connector := newTransactionTestConnector(t, repo, ownerConnector, snapshotRepo)

	// when: the calculated snapshot is persisted
	err = connector.snapshotEngineResult(t.Context(), snapshotParams{
		meter:                       meter.Meter{Aggregation: meter.MeterAggregationSum},
		owner:                       models.NamespacedID{Namespace: "ns", ID: "owner"},
		notAfter:                    start.Add(2 * time.Minute),
		snapshotInvalidationVersion: 4,
	}, engine.RunResult{History: history})

	// then: persistence still has the transaction that holds the owner lock
	require.NoError(t, err)
	require.True(t, driver.committed)
}
