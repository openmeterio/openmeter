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
	err = deletionService.DeleteEntitlement(t.Context(), namespace, ent.ID, effectiveAt.Add(time.Hour))
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
