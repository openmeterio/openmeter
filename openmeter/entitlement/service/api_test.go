package service_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

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

		// then both facade operations fail the precondition
		_, err := conn.GetCustomerEntitlement(t.Context(), entitlement.GetCustomerEntitlementInput{
			CustomerID:    customerID,
			EntitlementID: boolEnt.ID,
		})
		require.True(t, models.IsGenericPreConditionFailedError(err), "expected precondition failed error, got: %v", err)

		_, err = conn.ListCustomerEntitlements(t.Context(), entitlement.ListCustomerEntitlementsInput{
			CustomerID: customerID,
		})
		require.True(t, models.IsGenericPreConditionFailedError(err), "expected precondition failed error, got: %v", err)
	})
}
