package service_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCreateEntitlementWithGrants(t *testing.T) {
	namespace := "ns1"

	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	mtr, err := deps.meterService.CreateMeter(t.Context(), meter.CreateMeterInput{
		Namespace:     namespace,
		Name:          "Meter 1",
		Key:           "meter1",
		Description:   nil,
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "test",
		EventFrom:     nil,
		ValueProperty: lo.ToPtr("$.value"),
		GroupBy:       nil,
	})
	require.NoError(t, err)
	require.NotNil(t, mtr)

	// Create feature
	feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Name:      "feature1",
		Key:       "feature1",
		Namespace: namespace,
		MeterSlug: lo.ToPtr(mtr.Key),
	})
	require.NoError(t, err)
	require.NotNil(t, feat)

	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, "cust1", "Customer 1")

	t.Run("Should error if creating entitlement with grants and entitlement type is not metered", func(t *testing.T) {
		_, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			FeatureKey:       lo.ToPtr(feat.Key),
			UsageAttribution: cust.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeBoolean,
		}, []entitlement.CreateEntitlementGrantInputs{
			{
				CreateGrantInput: credit.CreateGrantInput{
					Amount:      100,
					Priority:    0,
					EffectiveAt: time.Now().Truncate(time.Minute).Add(time.Minute),
					Expiration:  nil,
				},
			},
		})
		require.ErrorAs(t, err, &entitlement.ErrEntitlementGrantsOnlySupportedForMeteredEntitlements)
	})

	var entId string

	t.Run("Should create entitlement with grants", func(t *testing.T) {
		ent, err := conn.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			FeatureKey:       lo.ToPtr(feat.Key),
			UsageAttribution: cust.GetUsageAttribution(),
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   time.Now(),
			})),
			EntitlementType: entitlement.EntitlementTypeMetered,
		}, []entitlement.CreateEntitlementGrantInputs{
			{
				CreateGrantInput: credit.CreateGrantInput{
					Amount:      100,
					Priority:    0,
					EffectiveAt: time.Now().Truncate(time.Minute).Add(time.Minute),
					Expiration:  nil,
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, ent)
		entId = ent.ID

		grants, err := deps.registry.MeteredEntitlement.ListEntitlementGrants(t.Context(), ent.Namespace, meteredentitlement.ListEntitlementGrantsParams{
			CustomerID:                ent.CustomerID,
			EntitlementIDOrFeatureKey: ent.ID,
			Page:                      pagination.NewPage(1, 100),
		})
		require.NoError(t, err)

		require.Len(t, grants.Items, 1)
		require.Equal(t, 100.0, grants.Items[0].Amount)
		require.Equal(t, uint8(0), grants.Items[0].Priority)
		require.Equal(t, ent.ID, grants.Items[0].EntitlementID)
	})

	t.Run("Should override entitlement with grants", func(t *testing.T) {
		ent, err := conn.OverrideEntitlement(t.Context(), cust.ID, entId, entitlement.CreateEntitlementInputs{
			Namespace:        namespace,
			FeatureKey:       lo.ToPtr(feat.Key),
			UsageAttribution: cust.GetUsageAttribution(),
			EntitlementType:  entitlement.EntitlementTypeMetered,
			UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   time.Now(),
			})),
		}, []entitlement.CreateEntitlementGrantInputs{
			{
				CreateGrantInput: credit.CreateGrantInput{
					Amount:      101,
					Priority:    0,
					EffectiveAt: time.Now().Truncate(time.Minute).Add(time.Minute),
					Expiration:  nil,
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, ent)
		entId = ent.ID

		grants, err := deps.registry.MeteredEntitlement.ListEntitlementGrants(t.Context(), ent.Namespace, meteredentitlement.ListEntitlementGrantsParams{
			CustomerID:                ent.CustomerID,
			EntitlementIDOrFeatureKey: ent.ID,
			Page:                      pagination.NewPage(1, 100),
		})
		require.NoError(t, err)

		require.Len(t, grants.Items, 1)
		require.Equal(t, 101.0, grants.Items[0].Amount)
		require.Equal(t, uint8(0), grants.Items[0].Priority)
		require.Equal(t, ent.ID, grants.Items[0].EntitlementID)
	})
}

func TestScheduling(t *testing.T) {
	namespace := "ns1"

	dummyAttribution := streaming.NewCustomerUsageAttribution(
		"01K3HJMFE6FW7PS470BYZ3NCR2",
		nil,
		[]string{
			"subject1",
		},
	)

	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	mtr, err := deps.meterService.CreateMeter(t.Context(), meter.CreateMeterInput{
		Namespace:     namespace,
		Name:          "Meter 1",
		Key:           "meter1",
		Description:   nil,
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "test",
		EventFrom:     nil,
		ValueProperty: lo.ToPtr("$.value"),
		GroupBy:       nil,
	})
	require.NoError(t, err)
	require.NotNil(t, mtr)

	// Create feature
	feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Name:      "feature1",
		Key:       "feature1",
		Namespace: namespace,
		MeterSlug: lo.ToPtr(mtr.Key),
	})
	require.NoError(t, err)
	require.NotEmpty(t, feat)

	tt := []struct {
		name string
		fn   func(t *testing.T, conn entitlement.Service, deps *dependencies)
	}{
		{
			name: "Should not allow scheduling via create",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()
				_, err := conn.CreateEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: dummyAttribution,
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						ActiveFrom:       lo.ToPtr(testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")),
					},
					nil,
				)
				assert.EqualError(t, err, "activeTo and activeFrom are not supported in CreateEntitlement")

				_, err = conn.CreateEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: dummyAttribution,
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						ActiveTo:         lo.ToPtr(testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")),
					},
					nil,
				)
				assert.EqualError(t, err, "activeTo and activeFrom are not supported in CreateEntitlement")
			},
		},
		{
			name: "Should fail scheduling is contradictory",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom := testutils.GetRFC3339Time(t, "2024-01-03T15:00:00Z")
				activeTo := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// From after To
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom),
						ActiveTo:   lo.ToPtr(activeTo),
					},
				)
				assert.EqualError(t, err, "validation error: ActiveTo cannot be before ActiveFrom")

				// Same value
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom),
						ActiveTo:   lo.ToPtr(activeFrom),
					},
				)
				// ActiveFrom and ActiveTo can be the same
				assert.NoError(t, err)

				// ActiveTo present but not ActiveFrom
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveTo: lo.ToPtr(activeTo),
					},
				)
				assert.EqualError(t, err, "validation error: ActiveFrom must be set if ActiveTo is set")
			},
		},
		{
			name: "Should allow scheduling entitlement if no entitlement is present for pair",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")
				activeTo := testutils.GetRFC3339Time(t, "2024-01-03T15:00:00Z")

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				ent, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr("feature1"),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom),
						ActiveTo:   lo.ToPtr(activeTo),
					},
				)
				assert.NoError(t, err)
				assert.NotNil(t, ent)
				assert.Equal(t, &activeFrom, ent.ActiveFrom)
				assert.Equal(t, &activeTo, ent.ActiveTo)
			},
		},
		{
			name: "Should allow scheduling entitlement after current scheduled entitlement",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")
				activeTo1 := testutils.GetRFC3339Time(t, "2024-01-03T15:00:00Z")

				activeFrom2 := testutils.GetRFC3339Time(t, "2024-01-03T18:00:00Z")
				activeTo2 := testutils.GetRFC3339Time(t, "2024-01-03T19:00:00Z")

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// Create first entitlement
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
						ActiveTo:   lo.ToPtr(activeTo1),
					},
				)
				assert.NoError(t, err)

				// Create second entitlement
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom2),
						ActiveTo:   lo.ToPtr(activeTo2),
					},
				)
				assert.NoError(t, err)
			},
		},
		{
			name: "Should error if entitlements with defined schedules overlap",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")
				activeTo1 := testutils.GetRFC3339Time(t, "2024-01-03T15:00:00Z")

				activeFrom2 := testutils.GetRFC3339Time(t, "2024-01-03T14:00:00Z")
				activeTo2 := testutils.GetRFC3339Time(t, "2024-01-03T16:00:00Z")

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
						ActiveTo:   lo.ToPtr(activeTo1),
					},
				)
				assert.NoError(t, err)

				// Create second entitlement
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom2),
						ActiveTo:   lo.ToPtr(activeTo2),
					},
				)

				var conflictErr *entitlement.AlreadyExistsError
				assert.ErrorAsf(t, err, &conflictErr, "expected error to be of type %T", conflictErr)
				assert.Equal(t, ent1.ID, conflictErr.EntitlementID)
			},
		},
		{
			name: "Should error when attempting to schedule after indefinite entitlement",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				activeFrom2 := testutils.GetRFC3339Time(t, "2024-01-03T14:00:00Z")
				activeTo2 := testutils.GetRFC3339Time(t, "2024-01-03T16:00:00Z")

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
					},
				)
				assert.Nil(t, err)

				// Create second entitlement
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom2),
						ActiveTo:   lo.ToPtr(activeTo2),
					},
				)

				var conflictErr *entitlement.AlreadyExistsError
				assert.ErrorAsf(t, err, &conflictErr, "expected error to be of type %T", conflictErr)
				assert.Equal(t, ent1.ID, conflictErr.EntitlementID)
			},
		},
		{
			name: "Should save annotations for all entitlement types",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				// create customer and subject
				randName1 := testutils.NameGenerator.Generate()
				cust1 := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName1.Key, randName1.Name)

				randName2 := testutils.NameGenerator.Generate()
				cust2 := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName2.Key, randName2.Name)

				randName3 := testutils.NameGenerator.Generate()
				cust3 := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName3.Key, randName3.Name)

				t.Run("Boolean entitlement", func(t *testing.T) {
					ent, err := conn.ScheduleEntitlement(
						ctx,
						entitlement.CreateEntitlementInputs{
							Namespace:        namespace,
							FeatureKey:       lo.ToPtr(feat.Key),
							UsageAttribution: cust1.GetUsageAttribution(),
							EntitlementType:  entitlement.EntitlementTypeBoolean,
							Annotations: models.Annotations{
								"subscription.id": "sub_123",
							},
						},
					)
					assert.NoError(t, err)
					assert.NotNil(t, ent)
					assert.Equal(t, models.Annotations{
						"subscription.id": "sub_123",
					}, ent.Annotations)
				})

				t.Run("Static entitlement", func(t *testing.T) {
					ent, err := conn.ScheduleEntitlement(
						ctx,
						entitlement.CreateEntitlementInputs{
							Namespace:        namespace,
							FeatureKey:       lo.ToPtr(feat.Key),
							UsageAttribution: cust2.GetUsageAttribution(),
							EntitlementType:  entitlement.EntitlementTypeStatic,
							Annotations: models.Annotations{
								"subscription.id": "sub_123",
							},
							Config: lo.ToPtr(`{"value": "100"}`),
						},
					)
					assert.NoError(t, err)
					assert.NotNil(t, ent)
					assert.Equal(t, models.Annotations{
						"subscription.id": "sub_123",
					}, ent.Annotations)
				})

				t.Run("Metered entitlement", func(t *testing.T) {
					ent, err := conn.ScheduleEntitlement(
						ctx,
						entitlement.CreateEntitlementInputs{
							Namespace:        namespace,
							FeatureKey:       lo.ToPtr(feat.Key),
							UsageAttribution: cust3.GetUsageAttribution(),
							EntitlementType:  entitlement.EntitlementTypeMetered,
							Annotations: models.Annotations{
								"subscription.id": "sub_123",
							},
							UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
								Interval: timeutil.RecurrencePeriodDaily,
								Anchor:   time.Now(),
							})),
						},
					)
					assert.NoError(t, err)
					assert.NotNil(t, ent)
					assert.Equal(t, models.Annotations{
						"subscription.id": "sub_123",
					}, ent.Annotations)
				})
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			tc.fn(t, conn, deps)
		})
	}
}

func TestSuperseding(t *testing.T) {
	namespace := "ns2"

	dummyAttribution := streaming.NewCustomerUsageAttribution(
		"01K3HJMFE6FW7PS470BYZ3NCR2",
		nil,
		[]string{
			"subject1",
		},
	)

	conn, deps := setupDependecies(t)
	defer deps.Teardown()

	// Create feature
	feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Name:      "feature1",
		Key:       "feature1",
		Namespace: namespace,
	})
	require.NoError(t, err)
	require.NotEmpty(t, feat)

	feat2, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Name:      "feature2",
		Key:       "feature2",
		Namespace: namespace,
	})
	require.NoError(t, err)
	require.NotEmpty(t, feat2)

	tt := []struct {
		name string
		fn   func(t *testing.T, conn entitlement.Service, deps *dependencies)
	}{
		{
			name: "Should error if original entitlement is not found",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				// Supersede nonexistent entitlement
				_, err = conn.SupersedeEntitlement(
					ctx,
					"bogus-id",
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: dummyAttribution,
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
					},
				)
				assert.EqualError(t, err, fmt.Sprintf("entitlement not found bogus-id in namespace %s", namespace))
			},
		},
		{
			name: "Should error if feature is not found",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
					},
				)
				require.NoError(t, err)
				require.NotNil(t, ent1)

				invalidFeatureKey := "invalid-feature-key"

				// Supersede entitlement
				_, err = conn.SupersedeEntitlement(
					ctx,
					ent1.ID,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(invalidFeatureKey), // invalid value
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1.Add(time.Hour)),
					},
				)
				assert.EqualError(t, err, fmt.Sprintf("feature not found: %s", invalidFeatureKey))
			},
		},
		{
			name: "Should error for differing feature",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
					},
				)
				assert.Nil(t, err)

				// Supersede entitlement
				_, err = conn.SupersedeEntitlement(
					ctx,
					ent1.ID,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat2.Key), // invalid value
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1.Add(time.Hour)),
					},
				)
				assert.EqualError(t, err, "validation error: old and new entitlements belong to different features")
			},
		},
		{
			name: "Should error for differing subjects",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				// create customer and subject
				randName1 := testutils.NameGenerator.Generate()
				cust1 := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName1.Key, randName1.Name)

				randName2 := testutils.NameGenerator.Generate()
				cust2 := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName2.Key, randName2.Name)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust1.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
					},
				)
				require.NoError(t, err)
				require.NotNil(t, ent1)

				// Supersede entitlement
				_, err = conn.SupersedeEntitlement(
					ctx,
					ent1.ID,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust2.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						ActiveFrom:       lo.ToPtr(activeFrom1.Add(time.Hour)),
					},
				)
				assert.EqualError(t, err, "validation error: old and new entitlements belong to different customers")
			},
		},
		{
			name: "Should supersede entitlement",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				// create customer and subject
				randName := testutils.NameGenerator.Generate()
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
					},
				)
				require.NoError(t, err)

				// Supersede entitlement
				ent2, err := conn.SupersedeEntitlement(
					ctx,
					ent1.ID,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						ActiveFrom:       lo.ToPtr(activeFrom1.Add(time.Hour)),
					},
				)
				require.NoError(t, err)

				ent1, err = conn.GetEntitlement(ctx, ent1.Namespace, ent1.ID)
				require.NoError(t, err)

				assert.Equal(t, lo.ToPtr(activeFrom1.Add(time.Hour)), ent1.ActiveTo)
				assert.Equal(t, lo.ToPtr(activeFrom1.Add(time.Hour)), ent2.ActiveFrom)
				assert.Nil(t, ent2.ActiveTo)
			},
		},
		{
			name: "Should error if entitlements are not continuous",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
						ActiveTo:   lo.ToPtr(activeFrom1.Add(time.Hour)),
					},
				)
				assert.Nil(t, err)

				// Supersede entitlement
				_, err = conn.SupersedeEntitlement(
					ctx,
					ent1.ID,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						ActiveFrom:       lo.ToPtr(activeFrom1.Add(time.Hour * 2)),
						ActiveTo:         lo.ToPtr(activeFrom1.Add(time.Hour * 3)),
					},
				)
				assert.EqualError(t, err, "validation error: new entitlement must be active before the old one ends")
			},
		},
		{
			name: "Should use current time for scheduling if activeFrom is not provided",
			fn: func(t *testing.T, conn entitlement.Service, deps *dependencies) {
				ctx := t.Context()

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-01T12:00:00Z")

				randName := testutils.NameGenerator.Generate()

				// create customer and subject
				cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
					},
				)
				assert.Nil(t, err)

				currentTime := testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z")
				clock.FreezeTime(currentTime)
				defer clock.UnFreeze()

				// Supersede entitlement
				ent2, err := conn.SupersedeEntitlement(
					ctx,
					ent1.ID,
					entitlement.CreateEntitlementInputs{
						Namespace:        namespace,
						FeatureKey:       lo.ToPtr(feat.Key),
						UsageAttribution: cust.GetUsageAttribution(),
						EntitlementType:  entitlement.EntitlementTypeBoolean,
					},
				)
				assert.Nil(t, err)

				ent1, err = conn.GetEntitlement(ctx, ent1.Namespace, ent1.ID)
				assert.Nil(t, err)

				// This assertions could fail if the test is slow
				assert.Equal(t, lo.ToPtr(currentTime), ent1.ActiveTo)
				assert.Equal(t, lo.ToPtr(currentTime), ent2.ActiveFrom)
				assert.Nil(t, ent2.ActiveTo)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			tc.fn(t, conn, deps)
		})
	}
}
