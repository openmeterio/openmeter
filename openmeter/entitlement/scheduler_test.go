package entitlement_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSchedulingConstraint(t *testing.T) {
	tt := []struct {
		name     string
		ents     []entitlement.Entitlement
		expected error
	}{
		{
			name:     "Should work on empty input",
			ents:     []entitlement.Entitlement{},
			expected: nil,
		},
		{
			name: "Should error if entitlements belong to different features",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					feature:   "feature1",
					subject:   "subject1",
					createdAt: "2021-01-01T00:00:00Z",
				}),
				getEnt(t, getEntInp{
					feature:   "feature2",
					subject:   "subject1",
					createdAt: "2021-01-01T00:00:00Z",
				}),
			},
			expected: fmt.Errorf("entitlements must belong to the same feature, found [feature1 feature2]"),
		},
		{
			name: "Should error if entitlements belong to different subjects",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					feature:   "feature1",
					subject:   "subject1",
					createdAt: "2021-01-01T00:00:00Z",
				}),
				getEnt(t, getEntInp{
					feature:   "feature1",
					subject:   "subject2",
					createdAt: "2021-01-01T00:00:00Z",
				}),
			},
			expected: fmt.Errorf("entitlements must belong to the same subject, found [subject1 subject2]"),
		},
		{
			name: "Should not error for single entitlement",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					feature:   "feature1",
					subject:   "subject1",
					createdAt: "2021-01-01T00:00:00Z",
				}),
			},
			expected: nil,
		},
		{
			name: "Should not error for non overlapping entitlements",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					feature:   "feature1",
					subject:   "subject1",
					createdAt: "2021-01-01T00:00:00Z",
					activeTo:  lo.ToPtr("2021-01-02T00:00:00Z"),
				}),
				getEnt(t, getEntInp{
					feature:   "feature1",
					subject:   "subject1",
					createdAt: "2021-01-02T00:00:00Z",
					deletedAt: lo.ToPtr("2021-01-03T00:00:00Z"),
				}),
				getEnt(t, getEntInp{
					feature:    "feature1",
					subject:    "subject1",
					createdAt:  "2021-01-02T00:00:00Z",
					activeFrom: lo.ToPtr("2021-01-03T00:00:00Z"),
					deletedAt:  lo.ToPtr("2021-01-04T00:00:00Z"),
				}),
				getEnt(t, getEntInp{
					feature:    "feature1",
					subject:    "subject1",
					createdAt:  "2021-01-02T00:00:00Z",
					activeFrom: lo.ToPtr("2021-01-04T00:00:00Z"),
					activeTo:   lo.ToPtr("2021-01-05T00:00:00Z"),
					deletedAt:  lo.ToPtr("2021-01-06T00:00:00Z"),
				}),
				getEnt(t, getEntInp{
					feature:    "feature1",
					subject:    "subject1",
					createdAt:  "2021-01-02T00:00:00Z",
					activeFrom: lo.ToPtr("2021-01-05T00:00:00Z"),
					activeTo:   lo.ToPtr("2021-01-06T00:00:00Z"),
				}),
				getEnt(t, getEntInp{
					feature:   "feature1",
					subject:   "subject1",
					createdAt: "2021-01-07T00:00:00Z",
				}),
			},
			expected: nil,
		},
		{
			name: "Should error for overlapping entitlements if one is indefinite",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					id:        "1",
					feature:   "feature1",
					subject:   "subject1",
					createdAt: "2021-01-01T00:00:00Z",
				}),
				getEnt(t, getEntInp{
					id:        "2",
					feature:   "feature1",
					subject:   "subject1",
					createdAt: "2021-01-02T00:00:00Z",
					deletedAt: lo.ToPtr("2021-01-03T00:00:00Z"),
				}),
			},
			expected: fmt.Errorf("constraint violated: 1 is active at the same time as 2"),
		},
		{
			name: "Should error for overlapping entitlements",
			ents: []entitlement.Entitlement{
				getEnt(t, getEntInp{
					id:        "5",
					feature:   "feature1",
					subject:   "subject1",
					createdAt: "2021-01-01T00:00:00Z",
					activeTo:  lo.ToPtr("2021-01-03T00:00:00Z"),
				}),
				getEnt(t, getEntInp{
					id:        "2",
					feature:   "feature1",
					subject:   "subject1",
					createdAt: "2021-01-02T00:00:00Z",
					deletedAt: lo.ToPtr("2021-01-03T00:00:00Z"),
				}),
			},
			expected: fmt.Errorf("constraint violated: 5 is active at the same time as 2"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := entitlement.ValidateUniqueConstraint(tc.ents)
			if tc.expected == nil {
				assert.Nil(t, err)
			} else {
				assert.EqualError(t, err, tc.expected.Error())
			}
		})
	}
}

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

type getEntInp struct {
	id         string
	feature    string
	subject    string
	createdAt  string
	activeFrom *string
	activeTo   *string
	deletedAt  *string
}

func getEnt(t *testing.T, inp getEntInp) entitlement.Entitlement {
	createdAt := testutils.GetRFC3339Time(t, inp.createdAt)

	ent := entitlement.Entitlement{
		GenericProperties: entitlement.GenericProperties{
			EntitlementType: entitlement.EntitlementTypeBoolean,
			FeatureKey:      inp.feature,
			SubjectKey:      inp.subject,
			ManagedModel: models.ManagedModel{
				CreatedAt: createdAt,
			},
		},
	}

	if inp.activeFrom != nil {
		ent.ActiveFrom = lo.ToPtr(testutils.GetRFC3339Time(t, *inp.activeFrom))
	}

	if inp.activeTo != nil {
		ent.ActiveTo = lo.ToPtr(testutils.GetRFC3339Time(t, *inp.activeTo))
	}

	if inp.deletedAt != nil {
		ent.DeletedAt = lo.ToPtr(testutils.GetRFC3339Time(t, *inp.deletedAt))
	}

	ent.ID = inp.id

	return ent
}
