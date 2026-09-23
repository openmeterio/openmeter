package service_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/engine"
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

		// then both facade operations conflict with the deleted state
		_, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: boolFeature.Key,
		})
		require.True(t, models.IsGenericConflictError(err), "expected conflict error, got: %v", err)

		_, err = conn.ListCustomerEntitlementAccess(t.Context(), entitlement.ListCustomerEntitlementAccessInput{
			CustomerID: customerID,
		})
		require.True(t, models.IsGenericConflictError(err), "expected conflict error, got: %v", err)
	})
}

func TestCustomerEntitlementHistoryAPI(t *testing.T) {
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	namespace := "ns-customer-entitlement-history"
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

	createMeteredEntitlement := func(t *testing.T, cust *customer.Customer, feat feature.Feature) *entitlement.Entitlement {
		t.Helper()

		ent, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			UsageAttribution: cust.GetUsageAttribution(),
			FeatureKey:       &feat.Key,
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			})),
			IssueAfterReset: lo.ToPtr(10.0),
		}, nil)
		require.NoError(t, err)

		return ent
	}

	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-1", "Customer 1")
	customerID := customer.CustomerID{Namespace: namespace, ID: cust.ID}

	// given a metered entitlement with usage in two consecutive hours and a boolean entitlement
	meteredEnt := createMeteredEntitlement(t, cust, createFeature(t, "metered"))

	boolFeature := createFeature(t, "boolean")
	boolEnt, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       &boolFeature.Key,
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	}, nil)
	require.NoError(t, err)

	deps.streamingConnector.AddSimpleEvent(mtr.Key, 1, now.Add(time.Minute))
	deps.streamingConnector.AddSimpleEvent(mtr.Key, 2, now.Add(time.Hour+time.Minute))

	// when two hours pass
	clock.SetTime(now.Add(2 * time.Hour))

	historyInput := func(entitlementID string) entitlement.GetCustomerEntitlementHistoryInput {
		return entitlement.GetCustomerEntitlementHistoryInput{
			CustomerID:    customerID,
			EntitlementID: entitlementID,
			WindowSize:    meter.WindowSizeHour,
		}
	}

	t.Run("should reject an incomplete input", func(t *testing.T) {
		input := historyInput(meteredEnt.ID)
		input.WindowSize = ""

		_, err := conn.GetCustomerEntitlementHistory(t.Context(), input)
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("should reject an unsupported window size", func(t *testing.T) {
		input := historyInput(meteredEnt.ID)
		input.WindowSize = meter.WindowSizeMinute

		_, err := conn.GetCustomerEntitlementHistory(t.Context(), input)
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("should report a missing customer as not found", func(t *testing.T) {
		input := historyInput(meteredEnt.ID)
		input.CustomerID = customer.CustomerID{Namespace: namespace, ID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2"}

		_, err := conn.GetCustomerEntitlementHistory(t.Context(), input)
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("should report a missing entitlement as not found", func(t *testing.T) {
		_, err := conn.GetCustomerEntitlementHistory(t.Context(), historyInput("01K5A4V2X8Q9Z7M3N6P1R4S8T3"))

		_, ok := lo.ErrorsAs[*entitlement.NotFoundError](err)
		require.True(t, ok, "expected entitlement not found error, got: %v", err)
	})

	t.Run("should report an entitlement of another customer as not found", func(t *testing.T) {
		other := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-2", "Customer 2")
		otherEnt := createMeteredEntitlement(t, other, createFeature(t, "metered-other"))

		_, err := conn.GetCustomerEntitlementHistory(t.Context(), historyInput(otherEnt.ID))

		_, ok := lo.ErrorsAs[*entitlement.NotFoundError](err)
		require.True(t, ok, "expected entitlement not found error, got: %v", err)
	})

	t.Run("should reject a non-metered entitlement", func(t *testing.T) {
		_, err := conn.GetCustomerEntitlementHistory(t.Context(), historyInput(boolEnt.ID))

		_, ok := lo.ErrorsAs[*entitlement.WrongTypeError](err)
		require.True(t, ok, "expected wrong type error, got: %v", err)
	})

	t.Run("should return the windowed and burndown history", func(t *testing.T) {
		history, err := conn.GetCustomerEntitlementHistory(t.Context(), historyInput(meteredEnt.ID))
		require.NoError(t, err)

		// then the windows cover each hour since the last reset
		require.GreaterOrEqual(t, len(history.Windows), 2)
		require.Equal(t, 3.0, lo.SumBy(history.Windows, func(w entitlement.BalanceHistoryWindow) float64 { return w.UsageInPeriod }))
		require.True(t, now.Equal(history.Windows[0].From))
		require.True(t, now.Add(time.Hour).Equal(history.Windows[0].To))
		require.Equal(t, 1.0, history.Windows[0].UsageInPeriod)
		require.Equal(t, 10.0, history.Windows[0].BalanceAtStart)
		require.True(t, now.Add(time.Hour).Equal(history.Windows[1].From))
		require.Equal(t, 2.0, history.Windows[1].UsageInPeriod)
		require.Equal(t, 9.0, history.Windows[1].BalanceAtStart)

		// then the burndown consumes the default grant
		segments := history.Burndown.Segments()
		require.NotEmpty(t, segments)
		require.Equal(t, 10.0, segments[0].BalanceAtStart.Balance())
		require.Equal(t, 7.0, segments[len(segments)-1].ApplyUsage().Balance())
		require.Equal(t, 3.0, lo.SumBy(segments, func(s engine.GrantBurnDownHistorySegment) float64 { return s.TotalUsage }))
	})

	t.Run("should reject a deleted customer", func(t *testing.T) {
		require.NoError(t, deps.customerService.DeleteCustomer(t.Context(), customerID))
		clock.SetTime(clock.Now().Add(time.Minute))

		_, err := conn.GetCustomerEntitlementHistory(t.Context(), historyInput(meteredEnt.ID))
		require.True(t, models.IsGenericConflictError(err), "expected conflict error, got: %v", err)
	})
}
