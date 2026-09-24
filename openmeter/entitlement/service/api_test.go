package service_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
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
	"github.com/openmeterio/openmeter/pkg/filter"
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
	boolEnt, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
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
		for name, input := range map[string]entitlement.GetCustomerEntitlementAccessInput{
			"no identifier":    {CustomerID: customerID, At: clock.Now()},
			"both identifiers": {CustomerID: customerID, FeatureKey: boolFeature.Key, EntitlementID: meteredEnt.ID, At: clock.Now()},
			"no time":          {CustomerID: customerID, FeatureKey: boolFeature.Key},
		} {
			_, err := conn.GetCustomerEntitlementAccess(t.Context(), input)
			require.True(t, models.IsGenericValidationError(err), "%s: expected validation error, got: %v", name, err)
		}
	})

	t.Run("Get should report a missing customer as not found", func(t *testing.T) {
		_, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customer.CustomerID{Namespace: namespace, ID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2"},
			FeatureKey: boolFeature.Key,
			At:         clock.Now(),
		})
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("Get should return the boolean entitlement value", func(t *testing.T) {
		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: boolFeature.Key,
			At:         clock.Now(),
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
			At:         clock.Now(),
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
			At:         clock.Now(),
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

	t.Run("Get by feature key should not resolve an entitlement ID", func(t *testing.T) {
		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: meteredEnt.ID,
			At:         clock.Now(),
		})
		require.NoError(t, err)
		require.IsType(t, &entitlement.NoAccessValue{}, access.Value)
	})

	t.Run("Get should return no access for a feature without an entitlement", func(t *testing.T) {
		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: "unknown-feature",
			At:         clock.Now(),
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
			At:         clock.Now(),
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

	t.Run("Get by ID should return the metered entitlement by ID", func(t *testing.T) {
		// given a metered entitlement with recorded usage
		// when its current value is requested by ID
		// then the result includes its feature and balance
		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID:    customerID,
			EntitlementID: meteredEnt.ID,
			At:            clock.Now(),
		})
		require.NoError(t, err)
		require.Equal(t, meteredFeature.Key, access.FeatureKey)

		value, ok := access.Value.(*meteredentitlement.MeteredEntitlementValue)
		require.True(t, ok, "expected metered value, got %T", access.Value)
		require.Equal(t, 9.0, value.Balance)
		require.Equal(t, 1.0, value.UsageInPeriod)
	})

	t.Run("Get by ID should evaluate the balance at the requested time", func(t *testing.T) {
		// given a usage event recorded after the requested evaluation time
		// when the entitlement is evaluated at that earlier time
		// then the balance is untouched
		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID:    customerID,
			EntitlementID: meteredEnt.ID,
			At:            now.Add(30 * time.Second),
		})
		require.NoError(t, err)

		value, ok := access.Value.(*meteredentitlement.MeteredEntitlementValue)
		require.True(t, ok, "expected metered value, got %T", access.Value)
		require.Equal(t, 10.0, value.Balance)
		require.Equal(t, 0.0, value.UsageInPeriod)
	})

	t.Run("Get by ID should return no access before the entitlement became active", func(t *testing.T) {
		// given a metered entitlement that becomes active after the requested time
		// when its earlier value is requested by ID
		// then access is denied while the metered type is retained
		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID:    customerID,
			EntitlementID: meteredEnt.ID,
			At:            now.Add(-time.Hour),
		})
		require.NoError(t, err)
		require.Equal(t, meteredFeature.Key, access.FeatureKey)
		require.Equal(t, entitlement.EntitlementTypeMetered, access.Type)
		require.IsType(t, &entitlement.NoAccessValue{}, access.Value)
	})

	t.Run("Get by ID should report an unknown entitlement ID as not found", func(t *testing.T) {
		// given an entitlement ID that does not exist
		// when its value is requested by ID
		// then the lookup reports not found
		_, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID:    customerID,
			EntitlementID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2",
			At:            clock.Now(),
		})
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("Get by ID should hide another customer's entitlement", func(t *testing.T) {
		// given another customer with its own boolean entitlement
		// when the first customer requests that entitlement by ID
		// then the response conceals the other customer's entitlement
		other := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-2", "Customer 2")
		otherEnt, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			UsageAttribution: other.GetUsageAttribution(),
			FeatureKey:       &boolFeature.Key,
			EntitlementType:  entitlement.EntitlementTypeBoolean,
		}, nil)
		require.NoError(t, err)

		_, err = conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID:    customerID,
			EntitlementID: otherEnt.ID,
			At:            clock.Now(),
		})
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("Get by ID should report a deleted entitlement as not found at any time", func(t *testing.T) {
		// given the boolean entitlement gets deleted and time moves past the deletion
		// when its value is requested before deletion or at the current time
		// then the ID lookup reports not found at both times
		beforeDeletion := clock.Now().Add(-time.Second)
		require.NoError(t, conn.DeleteEntitlement(t.Context(), namespace, boolEnt.ID, clock.Now()))
		clock.SetTime(clock.Now().Add(time.Minute))

		for name, at := range map[string]time.Time{"before deletion": beforeDeletion, "now": clock.Now()} {
			_, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
				CustomerID:    customerID,
				EntitlementID: boolEnt.ID,
				At:            at,
			})
			require.True(t, models.IsGenericNotFoundError(err), "%s: expected not found error, got: %v", name, err)
		}
	})

	t.Run("Get by feature key should return no access for a deleted metered entitlement before its deletion", func(t *testing.T) {
		// given the metered entitlement gets deleted and time moves past the deletion
		// when its feature is checked at a time before deletion
		// then the unavailable credit engine value denies access while retaining the type
		beforeDeletion := clock.Now().Add(-time.Second)
		require.NoError(t, conn.DeleteEntitlement(t.Context(), namespace, meteredEnt.ID, clock.Now()))
		clock.SetTime(clock.Now().Add(time.Minute))

		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: meteredFeature.Key,
			At:         beforeDeletion,
		})
		require.NoError(t, err)
		require.Equal(t, meteredFeature.Key, access.FeatureKey)
		require.Equal(t, entitlement.EntitlementTypeMetered, access.Type)
		require.IsType(t, &entitlement.NoAccessValue{}, access.Value)
	})

	t.Run("Get and List should reject a deleted customer", func(t *testing.T) {
		// given the customer gets deleted and time moves past the deletion
		// when its access is queried by feature, list, or entitlement ID
		// then every facade operation conflicts with the deleted state
		require.NoError(t, deps.customerService.DeleteCustomer(t.Context(), customerID))
		clock.SetTime(clock.Now().Add(time.Minute))

		_, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: boolFeature.Key,
			At:         clock.Now(),
		})
		require.True(t, models.IsGenericConflictError(err), "expected conflict error, got: %v", err)

		_, err = conn.ListCustomerEntitlementAccess(t.Context(), entitlement.ListCustomerEntitlementAccessInput{
			CustomerID: customerID,
		})
		require.True(t, models.IsGenericConflictError(err), "expected conflict error, got: %v", err)

		_, err = conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID:    customerID,
			EntitlementID: meteredEnt.ID,
			At:            clock.Now(),
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

	t.Run("should reject a range spanning more than the maximum windows", func(t *testing.T) {
		input := historyInput(meteredEnt.ID)
		input.From = lo.ToPtr(clock.Now().Add(-1001 * time.Hour))

		_, err := conn.GetCustomerEntitlementHistory(t.Context(), input)
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
		require.ErrorContains(t, err, "1000 windows")
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

func TestCustomerEntitlementAPI(t *testing.T) {
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	namespace := "ns-customer-entitlement-api"
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

	otherCust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-2", "Customer 2")

	// given a boolean and a metered entitlement on the customer, an expired one, and
	// a boolean entitlement on another customer for the same feature
	boolFeature := createFeature(t, "boolean")
	boolEnt, err := conn.CreateCustomerEntitlement(t.Context(), entitlement.CreateCustomerEntitlementInput{
		CustomerID: customerID,
		Entitlement: entitlement.CreateEntitlementInputs{
			FeatureKey:      &boolFeature.Key,
			EntitlementType: entitlement.EntitlementTypeBoolean,
		},
	})
	require.NoError(t, err)

	clock.SetTime(now.Add(time.Minute))

	meteredFeature := createFeature(t, "metered")
	meteredEnt, err := conn.CreateCustomerEntitlement(t.Context(), entitlement.CreateCustomerEntitlementInput{
		CustomerID: customerID,
		Entitlement: entitlement.CreateEntitlementInputs{
			FeatureID:       &meteredFeature.ID,
			EntitlementType: entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   now,
			})),
			IssueAfterReset: lo.ToPtr(10.0),
		},
	})
	require.NoError(t, err)

	expiredFeature := createFeature(t, "expired")
	_, err = conn.ScheduleEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       &expiredFeature.Key,
		EntitlementType:  entitlement.EntitlementTypeBoolean,
		ActiveFrom:       lo.ToPtr(clock.Now()),
		ActiveTo:         lo.ToPtr(clock.Now().Add(30 * time.Minute)),
	})
	require.NoError(t, err)

	otherEnt, err := conn.CreateCustomerEntitlement(t.Context(), entitlement.CreateCustomerEntitlementInput{
		CustomerID: customer.CustomerID{Namespace: namespace, ID: otherCust.ID},
		Entitlement: entitlement.CreateEntitlementInputs{
			FeatureKey:      &boolFeature.Key,
			EntitlementType: entitlement.EntitlementTypeBoolean,
		},
	})
	require.NoError(t, err)

	// when an hour passes, the expired entitlement is no longer active
	clock.SetTime(now.Add(time.Hour))

	t.Run("Get should reject an incomplete input", func(t *testing.T) {
		_, err := conn.GetCustomerEntitlement(t.Context(), entitlement.GetCustomerEntitlementInput{
			CustomerID: customerID,
		})
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("Get should report a missing customer as not found", func(t *testing.T) {
		_, err := conn.GetCustomerEntitlement(t.Context(), entitlement.GetCustomerEntitlementInput{
			CustomerID:    customer.CustomerID{Namespace: namespace, ID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2"},
			EntitlementID: boolEnt.ID,
		})
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("Get should return the entitlement by ID", func(t *testing.T) {
		ent, err := conn.GetCustomerEntitlement(t.Context(), entitlement.GetCustomerEntitlementInput{
			CustomerID:    customerID,
			EntitlementID: meteredEnt.ID,
		})
		require.NoError(t, err)
		require.Equal(t, meteredEnt.ID, ent.ID)
		require.Equal(t, cust.ID, ent.CustomerID)
		require.Equal(t, entitlement.EntitlementTypeMetered, ent.EntitlementType)
	})

	t.Run("Get should report a missing entitlement as not found", func(t *testing.T) {
		_, err := conn.GetCustomerEntitlement(t.Context(), entitlement.GetCustomerEntitlementInput{
			CustomerID:    customerID,
			EntitlementID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2",
		})
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("Get should not reveal another customer's entitlement", func(t *testing.T) {
		_, err := conn.GetCustomerEntitlement(t.Context(), entitlement.GetCustomerEntitlementInput{
			CustomerID:    customerID,
			EntitlementID: otherEnt.ID,
		})
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("List should return the active entitlements of the customer", func(t *testing.T) {
		result, err := conn.ListCustomerEntitlements(t.Context(), entitlement.ListCustomerEntitlementsInput{
			CustomerID: customerID,
			Page:       pagination.NewPage(1, 10),
		})
		require.NoError(t, err)
		require.Equal(t, 2, result.TotalCount)

		ids := lo.Map(result.Items, func(item entitlement.Entitlement, _ int) string { return item.ID })
		require.Equal(t, []string{boolEnt.ID, meteredEnt.ID}, ids)
	})

	t.Run("List should honor the sort order", func(t *testing.T) {
		result, err := conn.ListCustomerEntitlements(t.Context(), entitlement.ListCustomerEntitlementsInput{
			CustomerID: customerID,
			OrderBy:    entitlement.ListEntitlementsOrderByCreatedAt,
			Order:      sortx.OrderDesc,
			Page:       pagination.NewPage(1, 10),
		})
		require.NoError(t, err)

		ids := lo.Map(result.Items, func(item entitlement.Entitlement, _ int) string { return item.ID })
		require.Equal(t, []string{meteredEnt.ID, boolEnt.ID}, ids)
	})

	t.Run("List should paginate", func(t *testing.T) {
		result, err := conn.ListCustomerEntitlements(t.Context(), entitlement.ListCustomerEntitlementsInput{
			CustomerID: customerID,
			Page:       pagination.NewPage(2, 1),
		})
		require.NoError(t, err)
		require.Equal(t, 2, result.TotalCount)
		require.Len(t, result.Items, 1)
		require.Equal(t, meteredEnt.ID, result.Items[0].ID)
	})

	t.Run("List should filter by feature and type", func(t *testing.T) {
		for name, input := range map[string]entitlement.ListCustomerEntitlementsInput{
			"feature id":  {CustomerID: customerID, FeatureID: &filter.FilterULID{Eq: lo.ToPtr(meteredFeature.ID)}},
			"feature key": {CustomerID: customerID, FeatureKey: &filter.FilterString{In: &[]string{meteredFeature.Key, "unknown_feature"}}},
			"type":        {CustomerID: customerID, Type: &filter.FilterString{Eq: lo.ToPtr(string(entitlement.EntitlementTypeMetered))}},
			"type neq":    {CustomerID: customerID, Type: &filter.FilterString{Ne: lo.ToPtr(string(entitlement.EntitlementTypeBoolean))}},
		} {
			result, err := conn.ListCustomerEntitlements(t.Context(), input)
			require.NoError(t, err, name)
			require.Len(t, result.Items, 1, name)
			require.Equal(t, meteredEnt.ID, result.Items[0].ID, name)
		}
	})

	t.Run("List should reject an incomplete input", func(t *testing.T) {
		_, err := conn.ListCustomerEntitlements(t.Context(), entitlement.ListCustomerEntitlementsInput{
			CustomerID: customer.CustomerID{Namespace: namespace},
		})
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("List should report a missing customer as not found", func(t *testing.T) {
		_, err := conn.ListCustomerEntitlements(t.Context(), entitlement.ListCustomerEntitlementsInput{
			CustomerID: customer.CustomerID{Namespace: namespace, ID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2"},
		})
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("Get and List should reject a deleted customer", func(t *testing.T) {
		// given the customer gets deleted and time moves past the deletion
		require.NoError(t, deps.customerService.DeleteCustomer(t.Context(), customerID))
		clock.SetTime(clock.Now().Add(time.Minute))

		// then both facade operations conflict with the deleted state
		_, err := conn.GetCustomerEntitlement(t.Context(), entitlement.GetCustomerEntitlementInput{
			CustomerID:    customerID,
			EntitlementID: boolEnt.ID,
		})
		require.True(t, models.IsGenericConflictError(err), "expected conflict error, got: %v", err)

		_, err = conn.ListCustomerEntitlements(t.Context(), entitlement.ListCustomerEntitlementsInput{
			CustomerID: customerID,
		})
		require.True(t, models.IsGenericConflictError(err), "expected conflict error, got: %v", err)
	})
}

func TestCustomerEntitlementResetAPI(t *testing.T) {
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	namespace := "ns-customer-entitlement-reset"
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

	// given a metered entitlement with usage in the current period and a boolean entitlement
	meteredFeature := createFeature(t, "metered")
	meteredEnt := createMeteredEntitlement(t, cust, meteredFeature)

	boolFeature := createFeature(t, "boolean")
	boolEnt, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       &boolFeature.Key,
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	}, nil)
	require.NoError(t, err)

	deps.streamingConnector.AddSimpleEvent(mtr.Key, 3, now.Add(time.Minute))

	// when an hour passes
	clock.SetTime(now.Add(time.Hour))

	resetInput := func(entitlementID string) entitlement.ResetCustomerEntitlementUsageInput {
		return entitlement.ResetCustomerEntitlementUsageInput{
			CustomerID:    customerID,
			EntitlementID: entitlementID,
		}
	}

	meteredValue := func(t *testing.T) *meteredentitlement.MeteredEntitlementValue {
		t.Helper()

		access, err := conn.GetCustomerEntitlementAccess(t.Context(), entitlement.GetCustomerEntitlementAccessInput{
			CustomerID: customerID,
			FeatureKey: meteredFeature.Key,
			At:         clock.Now(),
		})
		require.NoError(t, err)

		value, ok := access.Value.(*meteredentitlement.MeteredEntitlementValue)
		require.True(t, ok, "expected metered value, got %T", access.Value)

		return value
	}

	t.Run("should reject an incomplete input", func(t *testing.T) {
		input := resetInput(meteredEnt.ID)
		input.EntitlementID = ""

		err := conn.ResetCustomerEntitlementUsage(t.Context(), input)
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("should reject a reset in the future", func(t *testing.T) {
		input := resetInput(meteredEnt.ID)
		input.EffectiveAt = lo.ToPtr(clock.Now().Add(time.Hour))

		err := conn.ResetCustomerEntitlementUsage(t.Context(), input)
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("should report a missing customer as not found", func(t *testing.T) {
		input := resetInput(meteredEnt.ID)
		input.CustomerID = customer.CustomerID{Namespace: namespace, ID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2"}

		err := conn.ResetCustomerEntitlementUsage(t.Context(), input)
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("should report a missing entitlement as not found", func(t *testing.T) {
		err := conn.ResetCustomerEntitlementUsage(t.Context(), resetInput("01K5A4V2X8Q9Z7M3N6P1R4S8T3"))

		_, ok := lo.ErrorsAs[*entitlement.NotFoundError](err)
		require.True(t, ok, "expected entitlement not found error, got: %v", err)
	})

	t.Run("should report an entitlement of another customer as not found", func(t *testing.T) {
		other := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-2", "Customer 2")
		otherEnt := createMeteredEntitlement(t, other, createFeature(t, "metered-other"))

		err := conn.ResetCustomerEntitlementUsage(t.Context(), resetInput(otherEnt.ID))

		_, ok := lo.ErrorsAs[*entitlement.NotFoundError](err)
		require.True(t, ok, "expected entitlement not found error, got: %v", err)
	})

	t.Run("should reject a non-metered entitlement", func(t *testing.T) {
		err := conn.ResetCustomerEntitlementUsage(t.Context(), resetInput(boolEnt.ID))

		_, ok := lo.ErrorsAs[*entitlement.WrongTypeError](err)
		require.True(t, ok, "expected wrong type error, got: %v", err)
	})

	t.Run("should start a new usage period", func(t *testing.T) {
		// given usage consumed the default grant in the current period
		before := meteredValue(t)
		require.Equal(t, 3.0, before.UsageInPeriod)
		require.Equal(t, 7.0, before.Balance)

		// when the usage is reset now
		require.NoError(t, conn.ResetCustomerEntitlementUsage(t.Context(), resetInput(meteredEnt.ID)))

		// then the new period starts without usage and the default grant is reissued
		after := meteredValue(t)
		require.Equal(t, 0.0, after.UsageInPeriod)
		require.Equal(t, 10.0, after.Balance)
		require.True(t, now.Add(time.Hour).Equal(after.StartOfPeriod), "expected period to start at %s, got %s", now.Add(time.Hour), after.StartOfPeriod)
	})

	t.Run("should reject a second reset at the same time", func(t *testing.T) {
		err := conn.ResetCustomerEntitlementUsage(t.Context(), resetInput(meteredEnt.ID))
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("should reject a deleted customer", func(t *testing.T) {
		require.NoError(t, deps.customerService.DeleteCustomer(t.Context(), customerID))
		clock.SetTime(clock.Now().Add(time.Minute))

		err := conn.ResetCustomerEntitlementUsage(t.Context(), resetInput(meteredEnt.ID))
		require.True(t, models.IsGenericConflictError(err), "expected conflict error, got: %v", err)
	})
}

func TestDeleteCustomerEntitlementAPI(t *testing.T) {
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	namespace := "ns-delete-customer-entitlement-api"
	now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")

	clock.SetTime(now)
	defer clock.ResetTime()

	createFeature := func(t *testing.T, key string) feature.Feature {
		t.Helper()

		feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
			Key:       key,
			Name:      key,
			Namespace: namespace,
		})
		require.NoError(t, err)

		return feat
	}

	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-1", "Customer 1")
	customerID := customer.CustomerID{Namespace: namespace, ID: cust.ID}

	otherCust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-2", "Customer 2")

	// given two boolean entitlements on the customer and one on another customer
	// for the same feature
	feat := createFeature(t, "boolean")
	ent, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       &feat.Key,
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	}, nil)
	require.NoError(t, err)

	keptFeature := createFeature(t, "kept")
	keptEnt, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       &keptFeature.Key,
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	}, nil)
	require.NoError(t, err)

	otherEnt, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: otherCust.GetUsageAttribution(),
		FeatureKey:       &feat.Key,
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	}, nil)
	require.NoError(t, err)

	t.Run("Delete should reject an incomplete input", func(t *testing.T) {
		err := conn.DeleteCustomerEntitlement(t.Context(), entitlement.DeleteCustomerEntitlementInput{
			CustomerID: customerID,
		})
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("Delete should report a missing customer as not found", func(t *testing.T) {
		err := conn.DeleteCustomerEntitlement(t.Context(), entitlement.DeleteCustomerEntitlementInput{
			CustomerID:    customer.CustomerID{Namespace: namespace, ID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2"},
			EntitlementID: ent.ID,
		})
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("Delete should report a missing entitlement as not found", func(t *testing.T) {
		err := conn.DeleteCustomerEntitlement(t.Context(), entitlement.DeleteCustomerEntitlementInput{
			CustomerID:    customerID,
			EntitlementID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2",
		})
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("Delete should not reveal another customer's entitlement", func(t *testing.T) {
		err := conn.DeleteCustomerEntitlement(t.Context(), entitlement.DeleteCustomerEntitlementInput{
			CustomerID:    customerID,
			EntitlementID: otherEnt.ID,
		})
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))

		// then the other customer's entitlement is left untouched
		_, err = conn.GetEntitlement(t.Context(), namespace, otherEnt.ID)
		require.NoError(t, err)
	})

	t.Run("Delete should soft delete the entitlement of the customer", func(t *testing.T) {
		// when the entitlement gets deleted
		require.NoError(t, conn.DeleteCustomerEntitlement(t.Context(), entitlement.DeleteCustomerEntitlementInput{
			CustomerID:    customerID,
			EntitlementID: ent.ID,
		}))

		// then it can no longer be resolved and the customer's other entitlement is kept
		_, err := conn.GetEntitlement(t.Context(), namespace, ent.ID)
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))

		ents, err := conn.GetEntitlementsOfCustomer(t.Context(), namespace, cust.ID, clock.Now())
		require.NoError(t, err)
		require.Equal(t, []string{keptEnt.ID}, lo.Map(ents, func(item entitlement.Entitlement, _ int) string { return item.ID }))

		// then deleting it again is reported as not found
		err = conn.DeleteCustomerEntitlement(t.Context(), entitlement.DeleteCustomerEntitlementInput{
			CustomerID:    customerID,
			EntitlementID: ent.ID,
		})
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("Delete should reject a deleted customer", func(t *testing.T) {
		// given the customer gets deleted and time moves past the deletion
		require.NoError(t, deps.customerService.DeleteCustomer(t.Context(), customerID))
		clock.SetTime(clock.Now().Add(time.Minute))

		// then the facade operation conflicts with the deleted state
		err := conn.DeleteCustomerEntitlement(t.Context(), entitlement.DeleteCustomerEntitlementInput{
			CustomerID:    customerID,
			EntitlementID: keptEnt.ID,
		})
		require.True(t, models.IsGenericConflictError(err), "expected conflict error, got: %v", err)
	})
}

func TestNamespaceEntitlementAPI(t *testing.T) {
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	namespace := "ns-namespace-entitlement-api"
	otherNamespace := "ns-namespace-entitlement-api-other"
	now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")

	clock.SetTime(now)
	defer clock.ResetTime()

	createFeature := func(t *testing.T, ns, key string) feature.Feature {
		t.Helper()

		feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
			Key:       key,
			Name:      key,
			Namespace: ns,
		})
		require.NoError(t, err)

		return feat
	}

	createBooleanEntitlement := func(t *testing.T, cust *customer.Customer, feat feature.Feature) *entitlement.Entitlement {
		t.Helper()

		ent, err := conn.CreateCustomerEntitlement(t.Context(), entitlement.CreateCustomerEntitlementInput{
			CustomerID: customer.CustomerID{Namespace: cust.Namespace, ID: cust.ID},
			Entitlement: entitlement.CreateEntitlementInputs{
				FeatureKey:      &feat.Key,
				EntitlementType: entitlement.EntitlementTypeBoolean,
			},
		})
		require.NoError(t, err)

		return ent
	}

	// given two customers with entitlements for the same feature, an expired
	// entitlement, and an entitlement in another namespace
	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-1", "Customer 1")
	otherCust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-2", "Customer 2")
	foreignCust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, otherNamespace, "cust-1", "Customer 1")

	feat := createFeature(t, namespace, "boolean")
	custEnt := createBooleanEntitlement(t, cust, feat)

	clock.SetTime(now.Add(time.Minute))

	otherCustEnt := createBooleanEntitlement(t, otherCust, feat)

	expiredFeature := createFeature(t, namespace, "expired")
	_, err := conn.ScheduleEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        namespace,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       &expiredFeature.Key,
		EntitlementType:  entitlement.EntitlementTypeBoolean,
		ActiveFrom:       lo.ToPtr(clock.Now()),
		ActiveTo:         lo.ToPtr(clock.Now().Add(30 * time.Minute)),
	})
	require.NoError(t, err)

	foreignEnt := createBooleanEntitlement(t, foreignCust, createFeature(t, otherNamespace, "boolean"))

	// when an hour passes, the expired entitlement is no longer active
	clock.SetTime(now.Add(time.Hour))

	t.Run("Get should return the entitlement by ID", func(t *testing.T) {
		ent, err := conn.GetEntitlementByID(t.Context(), entitlement.GetEntitlementByIDInput{
			Namespace:     namespace,
			EntitlementID: otherCustEnt.ID,
		})
		require.NoError(t, err)
		require.Equal(t, otherCustEnt.ID, ent.ID)
		require.Equal(t, otherCust.ID, ent.CustomerID)
	})

	t.Run("Get should reject an incomplete input", func(t *testing.T) {
		_, err := conn.GetEntitlementByID(t.Context(), entitlement.GetEntitlementByIDInput{
			Namespace: namespace,
		})
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("Get should report a missing entitlement as not found", func(t *testing.T) {
		_, err := conn.GetEntitlementByID(t.Context(), entitlement.GetEntitlementByIDInput{
			Namespace:     namespace,
			EntitlementID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2",
		})
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("Get should not reveal an entitlement of another namespace", func(t *testing.T) {
		_, err := conn.GetEntitlementByID(t.Context(), entitlement.GetEntitlementByIDInput{
			Namespace:     namespace,
			EntitlementID: foreignEnt.ID,
		})
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("List should return the active entitlements of every customer in the namespace", func(t *testing.T) {
		result, err := conn.ListNamespaceEntitlements(t.Context(), entitlement.ListNamespaceEntitlementsInput{
			Namespace: namespace,
			Page:      pagination.NewPage(1, 10),
		})
		require.NoError(t, err)
		require.Equal(t, 2, result.TotalCount)

		ids := lo.Map(result.Items, func(item entitlement.Entitlement, _ int) string { return item.ID })
		require.Equal(t, []string{custEnt.ID, otherCustEnt.ID}, ids)
	})

	t.Run("List should honor the sort order", func(t *testing.T) {
		result, err := conn.ListNamespaceEntitlements(t.Context(), entitlement.ListNamespaceEntitlementsInput{
			Namespace: namespace,
			OrderBy:   entitlement.ListEntitlementsOrderByCreatedAt,
			Order:     sortx.OrderDesc,
			Page:      pagination.NewPage(1, 10),
		})
		require.NoError(t, err)

		ids := lo.Map(result.Items, func(item entitlement.Entitlement, _ int) string { return item.ID })
		require.Equal(t, []string{otherCustEnt.ID, custEnt.ID}, ids)
	})

	t.Run("List should filter by customer", func(t *testing.T) {
		for name, tc := range map[string]struct {
			filter   filter.FilterULID
			expected string
		}{
			"eq":  {filter: filter.FilterULID{Eq: lo.ToPtr(otherCust.ID)}, expected: otherCustEnt.ID},
			"in":  {filter: filter.FilterULID{In: &[]string{cust.ID, foreignCust.ID}}, expected: custEnt.ID},
			"neq": {filter: filter.FilterULID{Ne: lo.ToPtr(cust.ID)}, expected: otherCustEnt.ID},
		} {
			result, err := conn.ListNamespaceEntitlements(t.Context(), entitlement.ListNamespaceEntitlementsInput{
				Namespace:  namespace,
				CustomerID: &tc.filter,
			})
			require.NoError(t, err, name)
			require.Len(t, result.Items, 1, name)
			require.Equal(t, tc.expected, result.Items[0].ID, name)
		}
	})

	t.Run("List should reject an incomplete input", func(t *testing.T) {
		_, err := conn.ListNamespaceEntitlements(t.Context(), entitlement.ListNamespaceEntitlementsInput{})
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})
}

func TestCustomerEntitlementOverrideAPI(t *testing.T) {
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	namespace := "ns-customer-entitlement-override-api"
	now := testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")

	clock.FreezeTime(now)
	defer clock.UnFreeze()

	createFeature := func(t *testing.T, key string) feature.Feature {
		t.Helper()

		feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
			Key:       key,
			Name:      key,
			Namespace: namespace,
		})
		require.NoError(t, err)

		return feat
	}

	createBooleanEntitlement := func(t *testing.T, customerID customer.CustomerID, feat feature.Feature) *entitlement.Entitlement {
		t.Helper()

		ent, err := conn.CreateCustomerEntitlement(t.Context(), entitlement.CreateCustomerEntitlementInput{
			CustomerID: customerID,
			Entitlement: entitlement.CreateEntitlementInputs{
				FeatureID:       &feat.ID,
				EntitlementType: entitlement.EntitlementTypeBoolean,
			},
		})
		require.NoError(t, err)

		return ent
	}

	overrideInput := func(customerID customer.CustomerID, entitlementID string, feat feature.Feature) entitlement.OverrideCustomerEntitlementInput {
		return entitlement.OverrideCustomerEntitlementInput{
			CustomerID:    customerID,
			EntitlementID: entitlementID,
			Entitlement: entitlement.CreateEntitlementInputs{
				FeatureID:       &feat.ID,
				EntitlementType: entitlement.EntitlementTypeBoolean,
			},
		}
	}

	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-1", "Customer 1")
	customerID := customer.CustomerID{Namespace: namespace, ID: cust.ID}

	otherCust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-2", "Customer 2")
	otherCustomerID := customer.CustomerID{Namespace: namespace, ID: otherCust.ID}

	overriddenFeature := createFeature(t, "overridden")
	deletedFeature := createFeature(t, "deleted")
	otherFeature := createFeature(t, "other")

	oldEnt := createBooleanEntitlement(t, customerID, overriddenFeature)
	deletedEnt := createBooleanEntitlement(t, customerID, deletedFeature)
	otherCustomerEnt := createBooleanEntitlement(t, otherCustomerID, otherFeature)

	require.NoError(t, conn.DeleteEntitlement(t.Context(), namespace, deletedEnt.ID, clock.Now()))

	clock.FreezeTime(now.Add(time.Minute))

	t.Run("should reject an incomplete input", func(t *testing.T) {
		input := overrideInput(customerID, "", overriddenFeature)

		_, err := conn.OverrideCustomerEntitlement(t.Context(), input)
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("should report a missing customer as not found", func(t *testing.T) {
		input := overrideInput(customer.CustomerID{Namespace: namespace, ID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2"}, oldEnt.ID, overriddenFeature)

		_, err := conn.OverrideCustomerEntitlement(t.Context(), input)
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("should report a missing entitlement as not found", func(t *testing.T) {
		_, err := conn.OverrideCustomerEntitlement(t.Context(), overrideInput(customerID, "01K5A4V2X8Q9Z7M3N6P1R4S8T2", overriddenFeature))
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("should report a deleted entitlement as not found", func(t *testing.T) {
		_, err := conn.OverrideCustomerEntitlement(t.Context(), overrideInput(customerID, deletedEnt.ID, deletedFeature))
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("should report an entitlement of another customer as not found", func(t *testing.T) {
		_, err := conn.OverrideCustomerEntitlement(t.Context(), overrideInput(customerID, otherCustomerEnt.ID, otherFeature))
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("should reject a different feature", func(t *testing.T) {
		_, err := conn.OverrideCustomerEntitlement(t.Context(), overrideInput(customerID, oldEnt.ID, otherFeature))
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("should replace the entitlement without a gap", func(t *testing.T) {
		// when the entitlement is overridden with one for the same feature
		newEnt, err := conn.OverrideCustomerEntitlement(t.Context(), overrideInput(customerID, oldEnt.ID, overriddenFeature))
		require.NoError(t, err)

		// then the new entitlement starts when the old one ends
		require.NotEqual(t, oldEnt.ID, newEnt.ID)
		require.Equal(t, overriddenFeature.ID, newEnt.FeatureID)
		require.Equal(t, cust.ID, newEnt.CustomerID)
		require.True(t, newEnt.ActiveFromTime().Equal(clock.Now()))

		ended, err := conn.GetEntitlement(t.Context(), namespace, oldEnt.ID)
		require.NoError(t, err)
		require.NotNil(t, ended.ActiveTo)
		require.True(t, ended.ActiveTo.Equal(clock.Now()))
		require.Nil(t, ended.DeletedAt)

		// and only the new entitlement is listed for the feature
		result, err := conn.ListCustomerEntitlements(t.Context(), entitlement.ListCustomerEntitlementsInput{
			CustomerID: customerID,
			FeatureID:  &filter.FilterULID{Eq: lo.ToPtr(overriddenFeature.ID)},
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		require.Equal(t, newEnt.ID, result.Items[0].ID)
	})

	t.Run("should reject an entitlement that already ended", func(t *testing.T) {
		// given the overridden entitlement ended a minute ago
		clock.FreezeTime(clock.Now().Add(time.Minute))

		// when it is overridden again, then the override is rejected
		_, err := conn.OverrideCustomerEntitlement(t.Context(), overrideInput(customerID, oldEnt.ID, overriddenFeature))
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("should reject a deleted customer", func(t *testing.T) {
		// given the customer gets deleted and time moves past the deletion
		require.NoError(t, deps.customerService.DeleteCustomer(t.Context(), otherCustomerID))
		clock.FreezeTime(clock.Now().Add(time.Minute))

		// then the override conflicts with the deleted state
		_, err := conn.OverrideCustomerEntitlement(t.Context(), overrideInput(otherCustomerID, otherCustomerEnt.ID, otherFeature))
		require.True(t, models.IsGenericConflictError(err), "expected conflict error, got: %v", err)
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
		require.True(t, models.IsGenericConflictError(err), "expected conflict error, got: %v", err)
	})
}

func TestCreateCustomerEntitlementGrantAPI(t *testing.T) {
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	namespace := "ns-create-customer-entitlement-grant-api"
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

	createMeteredEntitlement := func(t *testing.T, cust *customer.Customer, featureKey string) *entitlement.Entitlement {
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
		}, nil)
		require.NoError(t, err)

		return ent
	}

	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-1", "Customer 1")
	customerID := customer.CustomerID{Namespace: namespace, ID: cust.ID}

	meteredFeature := createFeature(t, "metered", lo.ToPtr(mtr.ID))
	meteredEnt := createMeteredEntitlement(t, cust, meteredFeature.Key)

	createInput := func(entitlementID string) entitlement.CreateCustomerEntitlementGrantInput {
		return entitlement.CreateCustomerEntitlementGrantInput{
			CustomerID:    customerID,
			EntitlementID: entitlementID,
			Grant: entitlement.CreateEntitlementGrantInputs{
				CreateGrantInput: credit.CreateGrantInput{
					Amount:           10,
					Priority:         3,
					EffectiveAt:      now.Add(time.Hour),
					Expiration:       &grant.ExpirationPeriod{Count: 1, Duration: grant.ExpirationPeriodDurationWeek},
					ResetMaxRollover: 10,
					Metadata:         map[string]string{"source": "promo"},
				},
			},
		}
	}

	t.Run("Create should reject an incomplete input", func(t *testing.T) {
		input := createInput(meteredEnt.ID)
		input.Grant.Amount = 0

		_, err := conn.CreateCustomerEntitlementGrant(t.Context(), input)
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("Create should issue a grant listed for the entitlement", func(t *testing.T) {
		// when a grant is issued for the metered entitlement
		created, err := conn.CreateCustomerEntitlementGrant(t.Context(), createInput(meteredEnt.ID))
		require.NoError(t, err)

		// then the grant is owned by the entitlement and carries the requested settings
		require.NotEmpty(t, created.ID)
		require.Equal(t, meteredEnt.ID, created.OwnerID)
		require.Equal(t, 10.0, created.Amount)
		require.Equal(t, uint8(3), created.Priority)
		require.True(t, now.Add(time.Hour).Equal(created.EffectiveAt), "unexpected effective at: %s", created.EffectiveAt)
		require.True(t, now.Add(time.Hour).AddDate(0, 0, 7).Equal(lo.FromPtr(created.ExpiresAt)), "unexpected expires at: %s", lo.FromPtr(created.ExpiresAt))
		require.Equal(t, map[string]string{"source": "promo"}, created.Metadata)

		// and it is part of the grant list of the entitlement
		grants, err := conn.ListCustomerEntitlementGrants(t.Context(), entitlement.ListCustomerEntitlementGrantsInput{
			CustomerID:    customerID,
			EntitlementID: meteredEnt.ID,
			Page:          pagination.NewPage(1, 10),
		})
		require.NoError(t, err)
		require.Equal(t, 1, grants.TotalCount)
		require.Equal(t, created.ID, grants.Items[0].ID)
	})

	t.Run("Create should reject a grant effective before the current usage period", func(t *testing.T) {
		input := createInput(meteredEnt.ID)
		input.Grant.EffectiveAt = now.Add(-time.Hour)

		_, err := conn.CreateCustomerEntitlementGrant(t.Context(), input)
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("Create should reject an entitlement that cannot own grants", func(t *testing.T) {
		// given a boolean entitlement of the customer
		booleanFeature := createFeature(t, "boolean", nil)
		booleanEnt, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			UsageAttribution: cust.GetUsageAttribution(),
			FeatureKey:       &booleanFeature.Key,
			EntitlementType:  entitlement.EntitlementTypeBoolean,
		}, nil)
		require.NoError(t, err)

		_, err = conn.CreateCustomerEntitlementGrant(t.Context(), createInput(booleanEnt.ID))
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.WrongTypeError{}))
	})

	t.Run("Create should report a missing entitlement as not found", func(t *testing.T) {
		_, err := conn.CreateCustomerEntitlementGrant(t.Context(), createInput("01K5A4V2X8Q9Z7M3N6P1R4S8T2"))
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("Create should not reveal another customer's entitlement", func(t *testing.T) {
		otherCust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-2", "Customer 2")
		otherEnt := createMeteredEntitlement(t, otherCust, meteredFeature.Key)

		_, err := conn.CreateCustomerEntitlementGrant(t.Context(), createInput(otherEnt.ID))
		require.ErrorAs(t, err, lo.ToPtr(&entitlement.NotFoundError{}))
	})

	t.Run("Create should reject a deleted customer", func(t *testing.T) {
		// given the customer gets deleted and time moves past the deletion
		require.NoError(t, deps.customerService.DeleteCustomer(t.Context(), customerID))
		clock.SetTime(clock.Now().Add(time.Minute))

		_, err := conn.CreateCustomerEntitlementGrant(t.Context(), createInput(meteredEnt.ID))
		require.True(t, models.IsGenericConflictError(err), "expected conflict error, got: %v", err)
	})
}

func TestGrantAPI(t *testing.T) {
	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	namespace := "ns-grant-api"
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

	createEntitlement := func(t *testing.T, cust *customer.Customer, featureKey string) *entitlement.Entitlement {
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
		}, nil)
		require.NoError(t, err)

		return ent
	}

	createGrant := func(t *testing.T, ent *entitlement.Entitlement, effectiveAt time.Time) string {
		t.Helper()

		g, err := deps.registry.MeteredEntitlement.CreateGrant(t.Context(), namespace, ent.CustomerID, ent.ID, meteredentitlement.CreateEntitlementGrantInputs{
			CreateGrantInput: credit.CreateGrantInput{Amount: 10, EffectiveAt: effectiveAt},
		})
		require.NoError(t, err)

		return g.ID
	}

	// given two customers with entitlements for two features and a grant effective
	// every minute, created in the order of their effective time
	featureA := createFeature(t, "feature-a")
	featureB := createFeature(t, "feature-b")

	cust1 := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-1", "Customer 1")
	cust2 := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust-2", "Customer 2")

	cust1A := createEntitlement(t, cust1, featureA.Key)
	cust1B := createEntitlement(t, cust1, featureB.Key)
	cust2A := createEntitlement(t, cust2, featureA.Key)

	grant1 := createGrant(t, cust1A, now)
	grant2 := createGrant(t, cust1A, now.Add(time.Minute))
	grant3 := createGrant(t, cust1B, now.Add(2*time.Minute))
	grant4 := createGrant(t, cust2A, now.Add(3*time.Minute))

	listInput := func() entitlement.ListNamespaceGrantsInput {
		return entitlement.ListNamespaceGrantsInput{
			Namespace: namespace,
			OrderBy:   grant.OrderByEffectiveAt,
			Order:     sortx.OrderAsc,
			Page:      pagination.NewPage(1, 10),
		}
	}

	grantIDs := func(grants []grant.Grant) []string {
		return lo.Map(grants, func(g grant.Grant, _ int) string { return g.ID })
	}

	t.Run("List should reject an invalid input", func(t *testing.T) {
		input := listInput()
		input.Page = pagination.Page{}

		_, err := conn.ListNamespaceGrants(t.Context(), input)
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("List should sort by effective time in both directions", func(t *testing.T) {
		result, err := conn.ListNamespaceGrants(t.Context(), listInput())
		require.NoError(t, err)
		require.Equal(t, []string{grant1, grant2, grant3, grant4}, grantIDs(result.Items))
		require.Equal(t, 4, result.TotalCount)

		input := listInput()
		input.Order = sortx.OrderDesc

		result, err = conn.ListNamespaceGrants(t.Context(), input)
		require.NoError(t, err)
		require.Equal(t, []string{grant4, grant3, grant2, grant1}, grantIDs(result.Items))
	})

	t.Run("List should return the requested page", func(t *testing.T) {
		input := listInput()
		input.OrderBy = grant.OrderByCreatedAt
		input.Order = sortx.OrderDesc
		input.Page = pagination.NewPage(2, 3)

		result, err := conn.ListNamespaceGrants(t.Context(), input)
		require.NoError(t, err)
		require.Equal(t, []string{grant1}, grantIDs(result.Items))
		require.Equal(t, 4, result.TotalCount)
	})

	t.Run("List should filter by the entitlement's customer and feature", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			modify func(*entitlement.ListNamespaceGrantsInput)
			want   []string
		}{
			{
				name:   "customer ID",
				modify: func(i *entitlement.ListNamespaceGrantsInput) { i.CustomerIDs = []string{cust2.ID} },
				want:   []string{grant4},
			},
			{
				name:   "feature ID",
				modify: func(i *entitlement.ListNamespaceGrantsInput) { i.FeatureIDsOrKeys = []string{featureB.ID} },
				want:   []string{grant3},
			},
			{
				name:   "feature key",
				modify: func(i *entitlement.ListNamespaceGrantsInput) { i.FeatureIDsOrKeys = []string{featureA.Key} },
				want:   []string{grant1, grant2, grant4},
			},
			{
				name: "customer ID and feature key",
				modify: func(i *entitlement.ListNamespaceGrantsInput) {
					i.CustomerIDs = []string{cust1.ID}
					i.FeatureIDsOrKeys = []string{featureA.Key}
				},
				want: []string{grant1, grant2},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				input := listInput()
				tc.modify(&input)

				result, err := conn.ListNamespaceGrants(t.Context(), input)
				require.NoError(t, err)
				require.Equal(t, tc.want, grantIDs(result.Items))
			})
		}
	})

	t.Run("Void should report a missing grant as not found", func(t *testing.T) {
		err := conn.VoidGrant(t.Context(), entitlement.VoidGrantInput{
			GrantID: models.NamespacedID{Namespace: namespace, ID: "01K5A4V2X8Q9Z7M3N6P1R4S8T2"},
		})
		require.True(t, models.IsGenericNotFoundError(err), "expected not found error, got: %v", err)
	})

	t.Run("Void should void the grant at the given time and reject voiding it again", func(t *testing.T) {
		clock.SetTime(now.Add(10 * time.Minute))
		voidedAt := now.Add(5 * time.Minute)

		err := conn.VoidGrant(t.Context(), entitlement.VoidGrantInput{
			GrantID: models.NamespacedID{Namespace: namespace, ID: grant2},
			At:      &voidedAt,
		})
		require.NoError(t, err)

		voided, err := deps.registry.GrantRepo.GetGrant(t.Context(), models.NamespacedID{Namespace: namespace, ID: grant2})
		require.NoError(t, err)
		require.NotNil(t, voided.VoidedAt)
		require.True(t, voidedAt.Equal(*voided.VoidedAt))

		err = conn.VoidGrant(t.Context(), entitlement.VoidGrantInput{
			GrantID: models.NamespacedID{Namespace: namespace, ID: grant2},
		})
		require.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
	})

	t.Run("Void should default to the current time", func(t *testing.T) {
		voidNow := now.Add(20 * time.Minute)
		clock.FreezeTime(voidNow)
		defer clock.UnFreeze()

		err := conn.VoidGrant(t.Context(), entitlement.VoidGrantInput{
			GrantID: models.NamespacedID{Namespace: namespace, ID: grant3},
		})
		require.NoError(t, err)

		voided, err := deps.registry.GrantRepo.GetGrant(t.Context(), models.NamespacedID{Namespace: namespace, ID: grant3})
		require.NoError(t, err)
		require.NotNil(t, voided.VoidedAt)
		require.True(t, voidNow.Equal(*voided.VoidedAt))
	})

	t.Run("List should include voided grants and the grants of deleted entitlements only on request", func(t *testing.T) {
		require.NoError(t, conn.DeleteEntitlement(t.Context(), namespace, cust2A.ID, clock.Now()))

		result, err := conn.ListNamespaceGrants(t.Context(), listInput())
		require.NoError(t, err)
		require.Equal(t, []string{grant1, grant2, grant3}, grantIDs(result.Items))

		input := listInput()
		input.IncludeDeleted = true

		result, err = conn.ListNamespaceGrants(t.Context(), input)
		require.NoError(t, err)
		require.Equal(t, []string{grant1, grant2, grant3, grant4}, grantIDs(result.Items))
	})
}
