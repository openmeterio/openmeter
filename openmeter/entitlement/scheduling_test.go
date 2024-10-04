package entitlement_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestScheduling(t *testing.T) {
	tt := []struct {
		name string
		fn   func(t *testing.T, conn entitlement.Connector, deps *dependencies)
	}{
		{
			name: "Should not allow scheduling via create",
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()
				_, err := conn.CreateEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						ActiveFrom:      lo.ToPtr(testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")),
					},
				)
				assert.EqualError(t, err, "activeTo and activeFrom are not supported in CreateEntitlement")

				_, err = conn.CreateEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						ActiveTo:        lo.ToPtr(testutils.GetRFC3339Time(t, "2025-01-01T00:00:00Z")),
					},
				)
				assert.EqualError(t, err, "activeTo and activeFrom are not supported in CreateEntitlement")
			},
		},
		{
			name: "Should fail scheduling is contradictory",
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom := testutils.GetRFC3339Time(t, "2024-01-03T15:00:00Z")
				activeTo := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				// Create feature
				_, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature1",
					Key:       "feature1",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				// From after To
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom),
						ActiveTo:   lo.ToPtr(activeTo),
					},
				)
				assert.EqualError(t, err, "ActiveTo must be after ActiveFrom")

				// Same value
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom),
						ActiveTo:   lo.ToPtr(activeFrom),
					},
				)
				assert.EqualError(t, err, "ActiveTo must be after ActiveFrom")

				// ActiveTo present but not ActiveFrom
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveTo: lo.ToPtr(activeTo),
					},
				)
				assert.EqualError(t, err, "ActiveFrom must be set if ActiveTo is set")
			},
		},
		{
			name: "Should allow scheduling entitlement if no entitlement is present for pair",
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")
				activeTo := testutils.GetRFC3339Time(t, "2024-01-03T15:00:00Z")

				// Create feature
				_, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature1",
					Key:       "feature1",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				ent, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom),
						ActiveTo:   lo.ToPtr(activeTo),
					},
				)
				assert.Nil(t, err)
				assert.NotNil(t, ent)
				assert.Equal(t, &activeFrom, ent.ActiveFrom)
				assert.Equal(t, &activeTo, ent.ActiveTo)
			},
		},
		{
			name: "Should allow scheduling entitlement after current scheduled entitlement",
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")
				activeTo1 := testutils.GetRFC3339Time(t, "2024-01-03T15:00:00Z")

				activeFrom2 := testutils.GetRFC3339Time(t, "2024-01-03T18:00:00Z")
				activeTo2 := testutils.GetRFC3339Time(t, "2024-01-03T19:00:00Z")

				// Create feature
				_, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature1",
					Key:       "feature1",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				// Create first entitlement
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
						ActiveTo:   lo.ToPtr(activeTo1),
					},
				)
				assert.Nil(t, err)

				// Create second entitlement
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom2),
						ActiveTo:   lo.ToPtr(activeTo2),
					},
				)
				assert.Nil(t, err)
			},
		},
		{
			name: "Should error if entitlements with defined schedules overlap",
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")
				activeTo1 := testutils.GetRFC3339Time(t, "2024-01-03T15:00:00Z")

				activeFrom2 := testutils.GetRFC3339Time(t, "2024-01-03T14:00:00Z")
				activeTo2 := testutils.GetRFC3339Time(t, "2024-01-03T16:00:00Z")

				// Create feature
				_, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature1",
					Key:       "feature1",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
						ActiveTo:   lo.ToPtr(activeTo1),
					},
				)
				assert.Nil(t, err)

				// Create second entitlement
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
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
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				activeFrom2 := testutils.GetRFC3339Time(t, "2024-01-03T14:00:00Z")
				activeTo2 := testutils.GetRFC3339Time(t, "2024-01-03T16:00:00Z")

				// Create feature
				_, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature1",
					Key:       "feature1",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
					},
				)
				assert.Nil(t, err)

				// Create second entitlement
				_, err = conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
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
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			conn, deps := setupDependecies(t)
			defer deps.Teardown()
			tc.fn(t, conn, deps)
		})
	}
}

func TestSuperseding(t *testing.T) {
	tt := []struct {
		name string
		fn   func(t *testing.T, conn entitlement.Connector, deps *dependencies)
	}{
		{
			name: "Should error if original entitlement is not found",
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				// Create feature
				_, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature1",
					Key:       "feature1",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				// Supersede nonexistent entitlement
				_, err = conn.SupersedeEntitlement(
					ctx,
					"bogus-id",
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
					},
				)
				assert.EqualError(t, err, "entitlement not found bogus-id in namespace ns1")
			},
		},
		{
			name: "Should error if feature is not found",
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				// Create feature
				_, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature1",
					Key:       "feature1",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
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
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature2"), // invlid value
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1.Add(time.Hour)),
					},
				)
				assert.EqualError(t, err, "feature not found: feature2")
			},
		},
		{
			name: "Should error for differing feature",
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				// Create feature
				_, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature1",
					Key:       "feature1",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				_, err = deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature2",
					Key:       "feature2",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
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
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature2"), // invalid value
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1.Add(time.Hour)),
					},
				)
				assert.EqualError(t, err, "Old and new entitlements belong to different features")
			},
		},
		{
			name: "Should error for differing subjects",
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				// Create feature
				_, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature1",
					Key:       "feature1",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
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
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject2", // invalid value
						EntitlementType: entitlement.EntitlementTypeBoolean,
						ActiveFrom:      lo.ToPtr(activeFrom1.Add(time.Hour)),
					},
				)
				assert.EqualError(t, err, "Old and new entitlements belong to different subjects")
			},
		},
		{
			name: "Should supersede entitlement",
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				// Create feature
				_, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature1",
					Key:       "feature1",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						// 12h in future
						ActiveFrom: lo.ToPtr(activeFrom1),
					},
				)
				assert.Nil(t, err)

				// Supersede entitlement
				ent2, err := conn.SupersedeEntitlement(
					ctx,
					ent1.ID,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						ActiveFrom:      lo.ToPtr(activeFrom1.Add(time.Hour)),
					},
				)
				assert.Nil(t, err)

				ent1, err = conn.GetEntitlement(ctx, ent1.Namespace, ent1.ID)
				assert.Nil(t, err)

				assert.Equal(t, lo.ToPtr(activeFrom1.Add(time.Hour)), ent1.ActiveTo)
				assert.Equal(t, lo.ToPtr(activeFrom1.Add(time.Hour)), ent2.ActiveFrom)
				assert.Nil(t, ent2.ActiveTo)
			},
		},
		{
			name: "Should error if entitlements are not overlapping",
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()

				clock.SetTime(testutils.GetRFC3339Time(t, "2024-01-03T00:00:00Z"))

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-03T12:00:00Z")

				// Create feature
				_, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature1",
					Key:       "feature1",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
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
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
						ActiveFrom:      lo.ToPtr(activeFrom1.Add(time.Hour * 2)),
						ActiveTo:        lo.ToPtr(activeFrom1.Add(time.Hour * 3)),
					},
				)
				assert.EqualError(t, err, "New entitlement must be active before the old one ends")
			},
		},
		{
			name: "Should use current time for scheduling if activeFrom is not provided",
			fn: func(t *testing.T, conn entitlement.Connector, deps *dependencies) {
				ctx := context.Background()

				activeFrom1 := testutils.GetRFC3339Time(t, "2024-01-01T12:00:00Z")

				// Create feature
				_, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
					Name:      "feature1",
					Key:       "feature1",
					Namespace: "ns1",
				})

				assert.Nil(t, err)

				// Create first entitlement
				ent1, err := conn.ScheduleEntitlement(
					ctx,
					entitlement.CreateEntitlementInputs{
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
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
						Namespace:       "ns1",
						FeatureKey:      lo.ToPtr("feature1"),
						SubjectKey:      "subject1",
						EntitlementType: entitlement.EntitlementTypeBoolean,
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
			conn, deps := setupDependecies(t)
			defer deps.Teardown()
			tc.fn(t, conn, deps)
		})
	}
}
