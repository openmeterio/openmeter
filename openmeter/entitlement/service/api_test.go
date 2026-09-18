package service_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCustomerEntitlementAccessAPI(t *testing.T) {
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	namespace := "ns-customer-entitlement-access"
	now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")

	clock.SetTime(now)
	defer clock.ResetTime()

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

	createFeature := func(t *testing.T, key string) feature.Feature {
		t.Helper()

		feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
			Key:       key,
			Name:      key,
			Namespace: namespace,
			MeterID:   lo.ToPtr(mtr.ID),
		})
		require.NoError(t, err)

		return feat
	}

	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-1", "Customer 1")
	customerID := customer.CustomerID{Namespace: namespace, ID: cust.ID}

	// given a boolean, a static, a metered and an already expired entitlement on the customer
	boolFeature := createFeature(t, "b-boolean")
	_, err = conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       &boolFeature.Key,
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	}, nil)
	require.NoError(t, err)

	staticFeature := createFeature(t, "a-static")
	_, err = conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       &staticFeature.Key,
		EntitlementType:  entitlement.EntitlementTypeStatic,
		Config:           lo.ToPtr(`{"value": 10}`),
	}, nil)
	require.NoError(t, err)

	meteredFeature := createFeature(t, "c-metered")
	meteredEnt, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       &meteredFeature.Key,
		EntitlementType:  entitlement.EntitlementTypeMetered,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily,
			Anchor:   now,
		})),
		IssueAfterReset: lo.ToPtr(10.0),
	}, nil)
	require.NoError(t, err)

	expiredFeature := createFeature(t, "d-expired")
	_, err = conn.ScheduleEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       &expiredFeature.Key,
		EntitlementType:  entitlement.EntitlementTypeBoolean,
		ActiveFrom:       lo.ToPtr(now),
		ActiveTo:         lo.ToPtr(now.Add(30 * time.Minute)),
	})
	require.NoError(t, err)

	deps.streamingConnector.AddSimpleEvent(mtr.Key, 1, now.Add(time.Minute))

	// when an hour passes, the expired entitlement is no longer active
	clock.SetTime(now.Add(time.Hour))

	t.Run("Get should reject an incomplete input", func(t *testing.T) {
		_, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
		})
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("Get should report a missing customer as not found", func(t *testing.T) {
		_, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customer.CustomerID{Namespace: namespace, ID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2"},
			FeatureKey: boolFeature.Key,
		})
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("Get should return the boolean entitlement value", func(t *testing.T) {
		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: boolFeature.Key,
		})
		require.NoError(t, err)
		require.Equal(t, boolFeature.Key, access.FeatureKey)
		require.IsType(t, &booleanentitlement.BooleanEntitlementValue{}, access.Value)
		require.True(t, access.Value.HasAccess())
	})

	t.Run("Get should return the static entitlement config", func(t *testing.T) {
		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: staticFeature.Key,
		})
		require.NoError(t, err)

		value, ok := access.Value.(*staticentitlement.StaticEntitlementValue)
		require.True(t, ok, "expected static value, got %T", access.Value)
		require.JSONEq(t, `{"value": 10}`, value.Config)
	})

	t.Run("Get should return the metered entitlement balance", func(t *testing.T) {
		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: meteredFeature.Key,
		})
		require.NoError(t, err)
		require.Equal(t, meteredFeature.Key, access.FeatureKey)

		value, ok := access.Value.(*meteredentitlement.MeteredEntitlementValue)
		require.True(t, ok, "expected metered value, got %T", access.Value)
		require.True(t, value.HasAccess())
		require.Equal(t, 9.0, value.Balance)
		require.Equal(t, 1.0, value.UsageInPeriod)
		require.Equal(t, 10.0, value.TotalAvailableGrantAmount)
		require.Len(t, value.GrantBalances, 1)
	})

	t.Run("Get should resolve the entitlement by ID as well", func(t *testing.T) {
		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: meteredEnt.ID,
		})
		require.NoError(t, err)
		require.IsType(t, &meteredentitlement.MeteredEntitlementValue{}, access.Value)
	})

	t.Run("Get should return no access for a feature without an entitlement", func(t *testing.T) {
		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: "unknown-feature",
		})
		require.NoError(t, err)
		require.Equal(t, "unknown-feature", access.FeatureKey)
		require.IsType(t, &entitlement.NoAccessValue{}, access.Value)
		require.False(t, access.Value.HasAccess())
	})

	t.Run("Get should return no access for an expired entitlement", func(t *testing.T) {
		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: expiredFeature.Key,
		})
		require.NoError(t, err)
		require.Equal(t, expiredFeature.Key, access.FeatureKey)
		require.False(t, access.Value.HasAccess())
	})

	t.Run("List should return active entitlements sorted by feature key", func(t *testing.T) {
		items, err := conn.ListCustomerEntitlementAccess(t.Context(), entitlement.ListCustomerEntitlementAccessInput{
			CustomerID: customerID,
		})
		require.NoError(t, err)

		keys := lo.Map(items, func(item entitlement.CustomerEntitlementAccess, _ int) string { return item.FeatureKey })
		require.Equal(t, []string{staticFeature.Key, boolFeature.Key, meteredFeature.Key}, keys)

		for _, item := range items {
			require.True(t, item.Value.HasAccess(), "expected access for %s", item.FeatureKey)
		}
	})

	t.Run("List should reject an incomplete input", func(t *testing.T) {
		_, err := conn.ListCustomerEntitlementAccess(t.Context(), entitlement.ListCustomerEntitlementAccessInput{
			CustomerID: customer.CustomerID{Namespace: namespace},
		})
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("List should report a missing customer as not found", func(t *testing.T) {
		_, err := conn.ListCustomerEntitlementAccess(t.Context(), entitlement.ListCustomerEntitlementAccessInput{
			CustomerID: customer.CustomerID{Namespace: namespace, ID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2"},
		})
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("Get and List should reject a deleted customer", func(t *testing.T) {
		// given the customer gets deleted and time moves past the deletion
		require.NoError(t, deps.customerService.DeleteCustomer(t.Context(), customerID))
		clock.SetTime(clock.Now().Add(time.Minute))

		// then both facade operations fail the precondition
		_, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: boolFeature.Key,
		})
		require.True(t, models.IsGenericPreConditionFailedError(err), "expected precondition failed error, got: %v", err)

		_, err = conn.ListCustomerEntitlementAccess(t.Context(), entitlement.ListCustomerEntitlementAccessInput{
			CustomerID: customerID,
		})
		require.True(t, models.IsGenericPreConditionFailedError(err), "expected precondition failed error, got: %v", err)
	})
}

func TestCustomerEntitlementGrantAPI(t *testing.T) {
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	namespace := "ns-customer-entitlement-grant-api"
	now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")

	clock.SetTime(now)
	defer clock.ResetTime()

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

	createFeature := func(t *testing.T, key string, meterID *string) feature.Feature {
		t.Helper()

		feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
			Key:       key,
			Name:      key,
			Namespace: namespace,
			MeterID:   meterID,
		})
		require.NoError(t, err)

		return feat
	}

	createMeteredEntitlement := func(t *testing.T, cust *customer.Customer, featureKey string, grants ...entitlement.CreateEntitlementGrantInputs) *entitlement.Entitlement {
		t.Helper()

		ent, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			UsageAttribution: cust.GetUsageAttribution(),
			FeatureKey:       &featureKey,
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			})),
		}, grants)
		require.NoError(t, err)

		return ent
	}

	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-1", "Customer 1")
	customerID := customer.CustomerID{Namespace: namespace, ID: cust.ID}

	// given a metered entitlement of the customer with three grants effective a minute
	// apart, the last of which expires and recurs
	meteredFeature := createFeature(t, "metered", lo.ToPtr(mtr.ID))
	meteredEnt := createMeteredEntitlement(t, cust, meteredFeature.Key, entitlement.CreateEntitlementGrantInputs{
		CreateGrantInput: credit.CreateGrantInput{Amount: 1, EffectiveAt: now},
	})

	secondGrant, err := deps.registry.MeteredEntitlement.CreateGrant(t.Context(), namespace, cust.ID, meteredEnt.ID, meteredentitlement.CreateEntitlementGrantInputs{
		CreateGrantInput: credit.CreateGrantInput{Amount: 2, EffectiveAt: now.Add(time.Minute)},
	})
	require.NoError(t, err)

	thirdGrant, err := deps.registry.MeteredEntitlement.CreateGrant(t.Context(), namespace, cust.ID, meteredEnt.ID, meteredentitlement.CreateEntitlementGrantInputs{
		CreateGrantInput: credit.CreateGrantInput{
			Amount:      3,
			Priority:    5,
			EffectiveAt: now.Add(2 * time.Minute),
			Expiration:  &grant.ExpirationPeriod{Count: 1, Duration: grant.ExpirationPeriodDurationMonth},
			Recurrence:  &timeutil.Recurrence{Interval: timeutil.RecurrencePeriodWeek, Anchor: now.Add(2 * time.Minute)},
			Metadata:    map[string]string{"source": "promo"},
		},
	})
	require.NoError(t, err)

	// given a boolean entitlement of the customer, which cannot own grants
	booleanFeature := createFeature(t, "boolean", nil)
	booleanEnt, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       &booleanFeature.Key,
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	}, nil)
	require.NoError(t, err)

	// given a metered entitlement on another customer
	otherCust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-2", "Customer 2")
	otherEnt := createMeteredEntitlement(t, otherCust, meteredFeature.Key)

	listInput := func() entitlement.ListCustomerEntitlementGrantsInput {
		return entitlement.ListCustomerEntitlementGrantsInput{
			CustomerID:    customerID,
			EntitlementID: meteredEnt.ID,
			OrderBy:       grant.OrderByEffectiveAt,
			Order:         sortx.OrderAsc,
			Page:          pagination.NewPage(1, 10),
		}
	}

	t.Run("List should reject an incomplete input", func(t *testing.T) {
		input := listInput()
		input.Page = pagination.Page{}

		_, err := conn.ListCustomerEntitlementGrants(t.Context(), input)
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("List should report a missing customer as not found", func(t *testing.T) {
		input := listInput()
		input.CustomerID = customer.CustomerID{Namespace: namespace, ID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2"}

		_, err := conn.ListCustomerEntitlementGrants(t.Context(), input)
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("List should report a missing entitlement as not found", func(t *testing.T) {
		input := listInput()
		input.EntitlementID = "01K5A4V2X8Q9Z7M3N6P1R4S8T2"

		_, err := conn.ListCustomerEntitlementGrants(t.Context(), input)
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("List should not reveal another customer's entitlement", func(t *testing.T) {
		input := listInput()
		input.EntitlementID = otherEnt.ID

		_, err := conn.ListCustomerEntitlementGrants(t.Context(), input)
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("List should return the grants of the entitlement in the requested order", func(t *testing.T) {
		grants, err := conn.ListCustomerEntitlementGrants(t.Context(), listInput())
		require.NoError(t, err)

		require.Equal(t, 3, grants.TotalCount)
		require.Equal(t, []float64{1, 2, 3}, lo.Map(grants.Items, func(g grant.Grant, _ int) float64 { return g.Amount }))

		for _, g := range grants.Items {
			require.Equal(t, meteredEnt.ID, g.OwnerID)
		}

		third := grants.Items[2]
		require.Equal(t, thirdGrant.ID, third.ID)
		require.Equal(t, uint8(5), third.Priority)
		require.Equal(t, &grant.ExpirationPeriod{Count: 1, Duration: grant.ExpirationPeriodDurationMonth}, third.Expiration)
		require.True(t, now.Add(2*time.Minute).AddDate(0, 1, 0).Equal(lo.FromPtr(third.ExpiresAt)), "unexpected expires at: %s", lo.FromPtr(third.ExpiresAt))
		require.NotNil(t, third.Recurrence)
		require.Equal(t, map[string]string{"source": "promo"}, third.Metadata)

		// when the order is reversed and the second page of size one is requested
		input := listInput()
		input.Order = sortx.OrderDesc
		input.Page = pagination.NewPage(2, 1)

		grants, err = conn.ListCustomerEntitlementGrants(t.Context(), input)
		require.NoError(t, err)

		// then the middle grant is returned while the count still covers every grant
		require.Equal(t, 3, grants.TotalCount)
		require.Len(t, grants.Items, 1)
		require.Equal(t, secondGrant.ID, grants.Items[0].ID)
	})

	t.Run("List should be empty for an entitlement that cannot own grants", func(t *testing.T) {
		input := listInput()
		input.EntitlementID = booleanEnt.ID

		grants, err := conn.ListCustomerEntitlementGrants(t.Context(), input)
		require.NoError(t, err)
		require.Equal(t, 0, grants.TotalCount)
		require.Empty(t, grants.Items)
	})

	t.Run("List should hide deleted grants unless asked for", func(t *testing.T) {
		// given the second grant got deleted a minute ago; no production path deletes a
		// single grant of a live entitlement, so the deletion is recorded directly
		require.NoError(t, deps.dbClient.Grant.UpdateOneID(secondGrant.ID).SetDeletedAt(clock.Now().Add(-time.Minute)).Exec(t.Context()))

		grants, err := conn.ListCustomerEntitlementGrants(t.Context(), listInput())
		require.NoError(t, err)
		require.Equal(t, 2, grants.TotalCount)
		require.Equal(t, []float64{1, 3}, lo.Map(grants.Items, func(g grant.Grant, _ int) float64 { return g.Amount }))

		input := listInput()
		input.IncludeDeleted = true

		grants, err = conn.ListCustomerEntitlementGrants(t.Context(), input)
		require.NoError(t, err)
		require.Equal(t, 3, grants.TotalCount)
		require.NotNil(t, grants.Items[1].DeletedAt)
	})

	t.Run("List should reject a deleted customer", func(t *testing.T) {
		// given the customer gets deleted and time moves past the deletion
		require.NoError(t, deps.customerService.DeleteCustomer(t.Context(), customerID))
		clock.SetTime(clock.Now().Add(time.Minute))

		_, err := conn.ListCustomerEntitlementGrants(t.Context(), listInput())
		require.True(t, models.IsGenericPreConditionFailedError(err), "expected precondition failed error, got: %v", err)
	})
}
