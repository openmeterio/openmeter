package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	credithook "github.com/openmeterio/openmeter/openmeter/credit/hook"
	db_grant "github.com/openmeterio/openmeter/openmeter/ent/db/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	entitlementservice "github.com/openmeterio/openmeter/openmeter/entitlement/service"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type failingPublisher struct {
	eventbus.Publisher
	err error
}

func (p failingPublisher) Publish(context.Context, marshaler.Event) error {
	return p.err
}

type blockingPreDeleteHook struct {
	models.NoopServiceHook[entitlement.Entitlement]
	entered chan<- struct{}
	release <-chan struct{}
}

func (h blockingPreDeleteHook) PreDelete(ctx context.Context, _ *entitlement.Entitlement) error {
	h.entered <- struct{}{}

	select {
	case <-h.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func createMeteredEntitlementWithGrant(t *testing.T, conn entitlement.Service, deps *dependencies, namespace string) (*entitlement.Entitlement, string, time.Time) {
	t.Helper()

	mtr, err := deps.meterService.CreateMeter(t.Context(), meter.CreateMeterInput{
		Namespace:     namespace,
		Name:          "Meter 1",
		Key:           "meter1",
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "test",
		ValueProperty: lo.ToPtr("$.value"),
	})
	require.NoError(t, err)
	createMeterInPG(t, deps.dbClient, mtr)

	feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Name:      "feature1",
		Key:       "feature1",
		Namespace: namespace,
		MeterID:   lo.ToPtr(mtr.ID),
	})
	require.NoError(t, err)

	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust1", "Customer 1")
	effectiveAt := time.Now().Truncate(time.Minute)
	ent, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		FeatureKey:       lo.ToPtr(feat.Key),
		UsageAttribution: cust.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily,
			Anchor:   effectiveAt,
		})),
	}, []entitlement.CreateEntitlementGrantInputs{{
		CreateGrantInput: credit.CreateGrantInput{
			Amount:      1,
			EffectiveAt: effectiveAt,
		},
	}})
	require.NoError(t, err)

	return ent, cust.ID, effectiveAt
}

// TestListEntitlementGrantsPagination pins that the caller controlled window is honored in both
// pagination modes. The HTTP layer picks one of the two, so silently falling back to the defaults
// makes large grant sets unreachable.
func TestListEntitlementGrantsPagination(t *testing.T) {
	namespace := "ns1"

	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	mtr, err := deps.meterService.CreateMeter(t.Context(), meter.CreateMeterInput{
		Namespace:     namespace,
		Name:          "Meter 1",
		Key:           "meter1",
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "test",
		ValueProperty: lo.ToPtr("$.value"),
	})
	require.NoError(t, err)
	createMeterInPG(t, deps.dbClient, mtr)

	feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Name:      "feature1",
		Key:       "feature1",
		Namespace: namespace,
		MeterID:   lo.ToPtr(mtr.ID),
	})
	require.NoError(t, err)

	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust1", "Customer 1")

	// given a metered entitlement with three grants, each effective a minute apart so that
	// ordering by effectiveAt is deterministic
	firstEffectiveAt := time.Now().Truncate(time.Minute).Add(time.Minute)

	ent, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		FeatureKey:       lo.ToPtr(feat.Key),
		UsageAttribution: cust.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily,
			Anchor:   time.Now(),
		})),
	}, []entitlement.CreateEntitlementGrantInputs{
		{CreateGrantInput: credit.CreateGrantInput{Amount: 1, EffectiveAt: firstEffectiveAt}},
	})
	require.NoError(t, err)

	for i := 2; i <= 3; i++ {
		_, err := deps.registry.MeteredEntitlement.CreateGrant(t.Context(), namespace, cust.ID, ent.ID, meteredentitlement.CreateEntitlementGrantInputs{
			CreateGrantInput: credit.CreateGrantInput{
				Amount:      float64(i),
				EffectiveAt: firstEffectiveAt.Add(time.Duration(i) * time.Minute),
			},
		})
		require.NoError(t, err)
	}

	listParams := func() meteredentitlement.ListEntitlementGrantsParams {
		return meteredentitlement.ListEntitlementGrantsParams{
			CustomerID:                cust.ID,
			EntitlementIDOrFeatureKey: ent.ID,
			OrderBy:                   grant.OrderByEffectiveAt,
			Order:                     sortx.OrderAsc,
		}
	}

	t.Run("Should return the requested page and the total count", func(t *testing.T) {
		// when the second page of size one is requested
		params := listParams()
		params.Page = pagination.NewPage(2, 1)

		grants, err := deps.registry.MeteredEntitlement.ListEntitlementGrants(t.Context(), namespace, params)
		require.NoError(t, err)

		// then only the second grant is returned, while the count still covers every grant
		require.Len(t, grants.Items, 1)
		require.Equal(t, 2.0, grants.Items[0].Amount)
		require.Equal(t, 3, grants.TotalCount)
		require.Equal(t, 2, grants.Page.PageNumber)
		require.Equal(t, 1, grants.Page.PageSize)
	})

	t.Run("Should return the requested limit and offset window", func(t *testing.T) {
		// when the deprecated limit/offset mode skips the first grant
		params := listParams()
		params.Limit = 2
		params.Offset = 1

		grants, err := deps.registry.MeteredEntitlement.ListEntitlementGrants(t.Context(), namespace, params)
		require.NoError(t, err)

		// then the remaining two grants are returned, and the count covers every match rather
		// than the window, so a caller walking the offset can tell when it is done
		require.Len(t, grants.Items, 2)
		require.Equal(t, 2.0, grants.Items[0].Amount)
		require.Equal(t, 3.0, grants.Items[1].Amount)
		require.Equal(t, 3, grants.TotalCount)
	})

	t.Run("Should order by id", func(t *testing.T) {
		// given ULIDs, which sort by creation time, ordering by id descending reverses the
		// order the grants were issued in
		params := listParams()
		params.OrderBy = grant.OrderByID
		params.Order = sortx.OrderDesc
		params.Page = pagination.NewPage(1, 100)

		// when the grants are listed
		grants, err := deps.registry.MeteredEntitlement.ListEntitlementGrants(t.Context(), namespace, params)
		require.NoError(t, err)

		// then the ordering is applied, rather than the query going out unordered because the
		// column has no mapping
		require.Len(t, grants.Items, 3)
		require.Equal(t, 3.0, grants.Items[0].Amount)
		require.Equal(t, 2.0, grants.Items[1].Amount)
		require.Equal(t, 1.0, grants.Items[2].Amount)
	})

	t.Run("Should reject an unbounded query", func(t *testing.T) {
		// given neither pagination mode, when the grants are listed
		_, err := deps.registry.MeteredEntitlement.ListEntitlementGrants(t.Context(), namespace, listParams())

		// then the request is rejected instead of returning every grant
		require.EqualError(t, err, "validation error: either page or limit is required")
	})

	t.Run("Should reject a negative offset", func(t *testing.T) {
		// given a bounded limit paired with a negative offset
		params := listParams()
		params.Limit = 1
		params.Offset = -1

		// when the grants are listed
		_, err := deps.registry.MeteredEntitlement.ListEntitlementGrants(t.Context(), namespace, params)

		// then the offset is rejected rather than silently ignored by the repository
		require.EqualError(t, err, "validation error: offset cannot be negative: -1")
	})
}

func TestDeleteEntitlementRollsBackGrantCleanup(t *testing.T) {
	// given: a metered entitlement with a grant and a publisher that fails after
	// the deletion hooks have run
	namespace := "ns-delete-rollback"
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	ent, _, effectiveAt := createMeteredEntitlementWithGrant(t, conn, deps, namespace)

	publishErr := errors.New("publish failed")
	deletionService := entitlementservice.NewEntitlementService(entitlementservice.ServiceConfig{
		EntitlementRepo: deps.registry.EntitlementRepo,
		CustomerService: deps.customerService,
		Publisher: failingPublisher{
			Publisher: eventbus.NewMock(t),
			err:       publishErr,
		},
	})
	deletionService.RegisterHooks(credithook.NewEntitlementHook(deps.registry.GrantRepo))

	// when: event publication makes the entitlement deletion roll back
	err := deletionService.DeleteEntitlement(t.Context(), namespace, ent.ID, effectiveAt.Add(time.Hour))
	require.ErrorIs(t, err, publishErr)

	// then: grant cleanup rolls back with the entitlement instead of committing independently
	entitlementRow, err := deps.dbClient.Entitlement.Get(t.Context(), ent.ID)
	require.NoError(t, err)
	require.Nil(t, entitlementRow.DeletedAt)

	grantRows, err := deps.dbClient.Grant.Query().
		Where(db_grant.Namespace(namespace), db_grant.OwnerID(ent.ID)).
		All(t.Context())
	require.NoError(t, err)
	require.Len(t, grantRows, 1)
	require.Nil(t, grantRows[0].DeletedAt)
}

func TestDeleteEntitlementSerializesGrantCreation(t *testing.T) {
	// given: deletion pauses after cleaning up an entitlement's existing grants
	namespace := "ns-delete-concurrent-grant"
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	ent, customerID, effectiveAt := createMeteredEntitlementWithGrant(t, conn, deps, namespace)

	deleteHookEntered := make(chan struct{}, 1)
	releaseDeleteHook := make(chan struct{})
	released := false
	defer func() {
		if !released {
			close(releaseDeleteHook)
		}
	}()

	deletionService := entitlementservice.NewEntitlementService(entitlementservice.ServiceConfig{
		EntitlementRepo: deps.registry.EntitlementRepo,
		CustomerService: deps.customerService,
		Publisher:       eventbus.NewMock(t),
	})
	deletionService.RegisterHooks(
		credithook.NewEntitlementHook(deps.registry.GrantRepo),
		blockingPreDeleteHook{entered: deleteHookEntered, release: releaseDeleteHook},
	)

	deleteDone := make(chan error, 1)
	go func() {
		deleteDone <- deletionService.DeleteEntitlement(t.Context(), namespace, ent.ID, effectiveAt)
	}()

	select {
	case <-deleteHookEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("deletion did not reach the blocking hook")
	}

	// when: another transaction tries to create a grant after cleanup but before deletion
	grantDone := make(chan error, 1)
	go func() {
		_, err := deps.registry.MeteredEntitlement.CreateGrant(t.Context(), namespace, customerID, ent.ID, meteredentitlement.CreateEntitlementGrantInputs{
			CreateGrantInput: credit.CreateGrantInput{
				Amount:      2,
				EffectiveAt: effectiveAt.Add(time.Minute),
			},
		})
		grantDone <- err
	}()

	var lockQueryErr error
	require.Eventually(t, func() bool {
		var blockedTransactions int
		lockQueryErr = deps.pgDriver.DB().QueryRowContext(t.Context(), `
			SELECT count(*)
			FROM pg_stat_activity
			WHERE datname = current_database()
				AND pid <> pg_backend_pid()
				AND wait_event_type = 'Lock'
		`).Scan(&blockedTransactions)

		return lockQueryErr == nil && blockedTransactions > 0
	}, 5*time.Second, 10*time.Millisecond, "grant creation did not wait for the entitlement lock")
	require.NoError(t, lockQueryErr)

	select {
	case err := <-grantDone:
		t.Fatalf("grant creation completed before deletion committed: %v", err)
	default:
	}

	close(releaseDeleteHook)
	released = true

	select {
	case err := <-deleteDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("deletion did not complete")
	}

	select {
	case err := <-grantDone:
		require.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("grant creation did not complete")
	}

	// then: no grant can be committed after the entitlement's cleanup
	liveGrantRows, err := deps.dbClient.Grant.Query().
		Where(
			db_grant.Namespace(namespace),
			db_grant.OwnerID(ent.ID),
			db_grant.DeletedAtIsNil(),
		).
		All(t.Context())
	require.NoError(t, err)
	require.Empty(t, liveGrantRows)
}
