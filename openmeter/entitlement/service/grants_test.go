package service_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

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

		// then the remaining two grants are returned; this mode carries no total count
		require.Len(t, grants.Items, 2)
		require.Equal(t, 2.0, grants.Items[0].Amount)
		require.Equal(t, 3.0, grants.Items[1].Amount)
	})

	t.Run("Should reject an unbounded query", func(t *testing.T) {
		// when neither pagination mode is provided the result set would be unbounded
		_, err := deps.registry.MeteredEntitlement.ListEntitlementGrants(t.Context(), namespace, listParams())
		require.EqualError(t, err, "either Page or Limit is required")
	})
}
