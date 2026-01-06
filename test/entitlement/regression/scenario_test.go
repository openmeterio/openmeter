package framework_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGrantExpiringAtReset(t *testing.T) {
	defer clock.ResetTime()
	deps := setupDependencies(t)
	defer deps.Close()
	ctx := context.Background()
	assert := assert.New(t)

	// Let's create a feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:30:21Z"))
	feature, err := deps.FeatureConnector.CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:      "feature-1",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feature)

	// create customer and subject
	cust := createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, "namespace-1", "subject-1", "Subject 1")

	// Let's create a new entitlement for the feature

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:35:21Z"))
	entitlement, err := deps.EntitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:        "namespace-1",
		FeatureID:        &feature.ID,
		FeatureKey:       &feature.Key,
		UsageAttribution: cust.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily,
			Anchor:   testutils.GetRFC3339Time(t, "2024-06-28T14:48:00Z"),
		})),
	}, nil)
	assert.NoError(err)
	assert.NotNil(entitlement)

	// Let's grant some credit

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:35:24Z"))
	grant1, err := deps.GrantConnector.CreateGrant(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		credit.CreateGrantInput{
			Amount:      10,
			Priority:    5,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-06-28T14:35:00Z"),
			Expiration: &grant.ExpirationPeriod{
				Count:    1,
				Duration: grant.ExpirationPeriodDurationYear,
			},
		})
	assert.NoError(err)
	assert.NotNil(grant1)

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:36:33Z"))
	grant2, err := deps.GrantConnector.CreateGrant(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		credit.CreateGrantInput{
			Amount:      20,
			Priority:    3,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-06-28T14:36:00Z"),
			Expiration: &grant.ExpirationPeriod{
				Count:    1,
				Duration: grant.ExpirationPeriodDurationDay,
			},
			ResetMaxRollover: 20,
		})
	assert.NoError(err)
	assert.NotNil(grant2)

	// Hack: this is in the future, but at least it won't return an error
	deps.Streaming.AddSimpleEvent("meter-1", 1, testutils.GetRFC3339Time(t, "2025-06-28T14:36:00Z"))

	// Let's query the usage
	currentBalance, err := deps.MeteredEntitlementConnector.GetEntitlementBalance(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		testutils.GetRFC3339Time(t, "2024-06-28T14:36:45Z"))
	assert.NoError(err)
	assert.NotNil(currentBalance)
	assert.Equal(30.0, currentBalance.Balance)

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-30T15:30:41Z"))
	// Let's query the usage
	currentBalance, err = deps.MeteredEntitlementConnector.GetEntitlementBalance(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		testutils.GetRFC3339Time(t, "2024-06-30T15:30:41Z"))
	assert.NoError(err)
	assert.NotNil(currentBalance)
	assert.Equal(0.0, currentBalance.Balance) // This test was previously faulty, grant2 has expired by this point while grant1 loses its balance on resets, two of which has already happened by this point

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T15:39:00Z"))
	grant3, err := deps.GrantConnector.CreateGrant(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		credit.CreateGrantInput{
			Amount:      100,
			Priority:    1,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-06-28T15:39:00Z"),
			Expiration: &grant.ExpirationPeriod{
				Count:    1,
				Duration: grant.ExpirationPeriodDurationYear,
			},
		})
	assert.NoError(err)
	assert.NotNil(grant3)

	// There should be a snapshot created
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-29T15:37:18Z"))
	reset, err := deps.MeteredEntitlementConnector.ResetEntitlementUsage(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		meteredentitlement.ResetEntitlementUsageParams{
			At:           testutils.GetRFC3339Time(t, "2024-06-29T14:48:00Z"),
			RetainAnchor: false,
		},
	)
	assert.NoError(err)
	assert.NotNil(reset)

	now := clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-30T15:42:41Z"))
	// Let's query the usage
	currentBalance, err = deps.MeteredEntitlementConnector.GetEntitlementBalance(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		now)
	assert.NoError(err)
	assert.NotNil(currentBalance)
	assert.Equal(0.0, currentBalance.Balance)
}

func TestGrantExpiringAndRecurringAtReset(t *testing.T) {
	defer clock.ResetTime()
	deps := setupDependencies(t)
	defer deps.Close()
	ctx := context.Background()
	assert := assert.New(t)

	// Hack: this is in the future, but at least it won't return an error
	deps.Streaming.AddSimpleEvent("meter-1", 1, testutils.GetRFC3339Time(t, "2025-06-28T14:36:00Z"))

	// Let's create a feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-02T08:43:52Z"))
	feature, err := deps.FeatureConnector.CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:      "feature-1",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feature)

	cust := createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, "namespace-1", "subject-1", "Subject 1")

	// Let's create a new entitlement for the feature

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-02T09:41:14Z"))
	entitlement, err := deps.EntitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:        "namespace-1",
		FeatureID:        &feature.ID,
		FeatureKey:       &feature.Key,
		UsageAttribution: cust.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily,
			Anchor:   testutils.GetRFC3339Time(t, "2024-07-02T09:41:00Z"),
		})),
	}, nil)
	assert.NoError(err)
	assert.NotNil(entitlement)

	// Let's grant some credit

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-02T09:43:04Z"))
	grant1, err := deps.GrantConnector.CreateGrant(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		credit.CreateGrantInput{
			Amount:           20,
			ResetMaxRollover: 20,
			Priority:         1,
			EffectiveAt:      testutils.GetRFC3339Time(t, "2024-07-02T09:43:00Z"),
			Expiration: &grant.ExpirationPeriod{
				Count:    1,
				Duration: grant.ExpirationPeriodDurationWeek,
			},
			Recurrence: &timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodDaily,
				Anchor:   testutils.GetRFC3339Time(t, "2024-07-02T09:43:00Z"),
			},
		})
	assert.NoError(err)
	assert.NotNil(grant1)

	// Let's reset as scheduled by entitlement (last reset before grant expiring)
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T09:41:00Z"))
	resetCommand := meteredentitlement.ResetEntitlementUsageParams{
		At:           testutils.GetRFC3339Time(t, "2024-07-09T09:41:00Z"),
		RetainAnchor: true,
	}
	reset, err := deps.MeteredEntitlementConnector.ResetEntitlementUsage(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		resetCommand,
	)
	assert.NoError(err)
	assert.NotNil(reset)

	// Let's query the usage after grant has expired
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T22:41:00Z"))
	currentBalance, err := deps.MeteredEntitlementConnector.GetEntitlementBalance(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		testutils.GetRFC3339Time(t, "2024-07-09T22:41:00Z"))
	assert.NoError(err)
	assert.NotNil(currentBalance)
	assert.Equal(0.0, currentBalance.Balance)

	// Let's query the usage again after snapshot exists
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-10T07:33:06Z"))
	// clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-10T09:41:00Z"))
	currentBalance, err = deps.MeteredEntitlementConnector.GetEntitlementBalance(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		testutils.GetRFC3339Time(t, "2024-07-10T07:33:06Z"))
	assert.NoError(err)
	assert.NotNil(currentBalance)
	assert.Equal(0.0, currentBalance.Balance)
}

func TestBalanceCalculationsAfterVoiding(t *testing.T) {
	defer clock.ResetTime()
	deps := setupDependencies(t)
	defer deps.Close()
	ctx := context.Background()
	assert := assert.New(t)

	// Let's create a feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-07T14:44:19Z"))
	feature, err := deps.FeatureConnector.CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:      "feature-1",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feature)

	cust := createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, "namespace-1", "subject-1", "Subject 1")

	// Let's create a new entitlement for the feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T11:20:28Z"))
	entitlement, err := deps.EntitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:        "namespace-1",
		FeatureID:        &feature.ID,
		FeatureKey:       &feature.Key,
		UsageAttribution: cust.GetUsageAttribution(),
		IssueAfterReset:  convert.ToPointer(500.0),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   testutils.GetRFC3339Time(t, "2024-07-01T00:00:00Z"),
		})),
	}, nil)
	assert.NoError(err)
	assert.NotNil(entitlement)

	// Let's retrieve the grant so we can reference it
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T12:20:28Z"))
	grants, err := deps.GrantRepo.ListGrants(ctx, grant.ListParams{
		Namespace:      "namespace-1",
		IncludeDeleted: true,
		Page: pagination.Page{
			PageSize:   100,
			PageNumber: 1,
		},
		OrderBy: grant.OrderByCreatedAt,
	})
	assert.NoError(err)
	assert.Len(grants.Items, 1)

	// Let's create another grant
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T12:09:40Z"))
	grant2, err := deps.GrantConnector.CreateGrant(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		credit.CreateGrantInput{
			Amount:      10000,
			Priority:    1,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-07-09T12:09:00Z"),
			Expiration: &grant.ExpirationPeriod{
				Count:    1,
				Duration: grant.ExpirationPeriodDurationWeek,
			},
		})
	assert.NoError(err)
	assert.NotNil(grant2)

	// Let's add some usage

	// Due to grant priority this usage should be deduceted from grant 2
	// The reason is that the default recurring grant's expiration date is way later (with hack set to 100y in future)
	deps.Streaming.AddSimpleEvent("meter-1", 10, testutils.GetRFC3339Time(t, "2024-07-09T13:09:00Z"))

	// As this is later, this usage should be deducted from grant 1

	// Lets void the grant
	voidTime := testutils.GetRFC3339Time(t, "2024-07-09T14:54:04Z")

	deps.Streaming.AddSimpleEvent("meter-1", 12, voidTime.Add(time.Minute*15))

	clock.SetTime(voidTime)
	err = deps.GrantConnector.VoidGrant(ctx, models.NamespacedID{
		Namespace: "namespace-1",
		ID:        grant2.ID,
	})
	assert.NoError(err)

	// Let's query the usage
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T16:38:00Z"))
	currentBalance, err := deps.MeteredEntitlementConnector.GetEntitlementBalance(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		testutils.GetRFC3339Time(t, "2024-07-09T16:38:00Z"))
	assert.NoError(err)
	assert.NotNil(currentBalance)
	assert.Equal(488.0, currentBalance.Balance)
}

func TestCreatingEntitlementsForKeyOfArchivedFeatures(t *testing.T) {
	defer clock.ResetTime()
	deps := setupDependencies(t)
	defer deps.Close()
	ctx := context.Background()
	assert := assert.New(t)

	// Let's create a feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-07T14:44:19Z"))
	feat, err := deps.FeatureConnector.CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:      "feature-1",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feat)

	cust := createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, "namespace-1", "subject-1", "Subject 1")

	// Let's create a new entitlement for the feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T11:20:28Z"))
	ent, err := deps.EntitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:        "namespace-1",
		FeatureID:        &feat.ID,
		FeatureKey:       &feat.Key,
		UsageAttribution: cust.GetUsageAttribution(),
		IssueAfterReset:  convert.ToPointer(500.0),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   testutils.GetRFC3339Time(t, "2024-07-01T00:00:00Z"),
		})),
	}, nil)
	assert.NoError(err)
	assert.NotNil(ent)

	// Let's archive the feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T12:20:28Z"))
	err = deps.FeatureConnector.ArchiveFeature(ctx, models.NamespacedID{
		Namespace: "namespace-1",
		ID:        feat.ID,
	})
	assert.NoError(err)

	// Let's create a new feature with the same key
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T13:20:28Z"))
	feature2, err := deps.FeatureConnector.CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:      "feature-1-2",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feature2)

	cust2 := createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, "namespace-1", "subject-2", "Subject 2")

	// Let's create a new entitlement for feature2 for subject-2
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T14:20:28Z"))
	ent2, err := deps.EntitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:        "namespace-1",
		FeatureID:        &feature2.ID,
		FeatureKey:       &feature2.Key,
		UsageAttribution: cust2.GetUsageAttribution(),
		IssueAfterReset:  convert.ToPointer(500.0),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   testutils.GetRFC3339Time(t, "2024-07-01T00:00:00Z"),
		})),
	}, nil)
	assert.NoError(err)
	assert.NotNil(ent2)
}

func TestGrantingAfterOverage(t *testing.T) {
	defer clock.ResetTime()
	deps := setupDependencies(t)
	defer deps.Close()
	ctx := context.Background()
	assert := assert.New(t)

	// Let's create a feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-07T14:44:19Z"))
	feature, err := deps.FeatureConnector.CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:      "feature-1",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feature)

	cust := createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, "namespace-1", "subject-1", "Subject 1")

	// Let's create a new entitlement for the feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-08-22T11:25:00Z"))
	ent, err := deps.EntitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:        "namespace-1",
		FeatureID:        &feature.ID,
		FeatureKey:       &feature.Key,
		UsageAttribution: cust.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   testutils.GetRFC3339Time(t, "2024-08-22T11:25:00Z"),
		})),
	}, nil)
	assert.NoError(err)
	assert.NotNil(ent)

	// Lets grant some credit for 500
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-08-22T11:27:18Z"))
	grant1, err := deps.GrantConnector.CreateGrant(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        ent.ID,
		},
		credit.CreateGrantInput{
			Amount:      500,
			Priority:    1,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-08-22T11:25:00Z"),
			Expiration: &grant.ExpirationPeriod{
				Count:    1,
				Duration: grant.ExpirationPeriodDurationMonth,
			},
			Recurrence: &timeutil.Recurrence{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   testutils.GetRFC3339Time(t, "2024-08-22T11:25:00Z"),
			},
		})
	assert.NoError(err)
	assert.NotNil(grant1)

	addInMany := func(amount int, from, to time.Time) {
		for i := 0; i < amount; i++ {
			dur := to.Sub(from)
			clock.SetTime(from.Add(time.Duration(i) * dur / time.Duration(amount)))

			deps.Streaming.AddSimpleEvent("meter-1", 1, clock.Now())
		}
	}

	// Lets register usage until it reaches overage
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-08-22T11:30:18Z"))
	addInMany(1000, testutils.GetRFC3339Time(t, "2024-08-22T11:30:18Z"), testutils.GetRFC3339Time(t, "2024-08-22T12:05:18Z"))

	// Lets grant more credits
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-08-22T12:31:30Z"))
	grant2, err := deps.GrantConnector.CreateGrant(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        ent.ID,
		},
		credit.CreateGrantInput{
			Amount:      8000,
			Priority:    1,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-08-22T12:31:00Z"),
			Expiration: &grant.ExpirationPeriod{
				Count:    1,
				Duration: grant.ExpirationPeriodDurationMonth,
			},
		})
	assert.NoError(err)
	assert.NotNil(grant2)

	// Lets register usage until it reaches overage
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-08-22T11:30:18Z"))
	deps.Streaming.AddSimpleEvent("meter-1", 1000, testutils.GetRFC3339Time(t, "2024-08-22T12:35:18Z"))

	// Lets get the balance
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-08-22T13:30:18Z"))
	currentBalance, err := deps.MeteredEntitlementConnector.GetEntitlementBalance(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        ent.ID,
		},
		testutils.GetRFC3339Time(t, "2024-08-22T13:30:18Z"))

	assert.NoError(err)
	assert.NotNil(currentBalance)
	assert.Equal(6500.0, currentBalance.Balance)
	assert.Equal(0.0, currentBalance.Overage)
	assert.Equal(2000.0, currentBalance.UsageInPeriod)
}

func TestBalanceWorkerActiveToFromEntitlementsMapping(t *testing.T) {
	defer clock.ResetTime()
	deps := setupDependencies(t)
	defer deps.Close()
	ctx := context.Background()
	assert := assert.New(t)

	// Let's create a feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-07T14:44:19Z"))
	feature, err := deps.FeatureConnector.CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:      "feature-1",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feature)

	cust := createCustomerAndSubject(t, deps.SubjectService, deps.CustomerService, "namespace-1", "subject-1", "Subject 1")

	// Let's create a new entitlement for the feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-08-22T11:25:00Z"))
	ent1, err := deps.EntitlementConnector.ScheduleEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:        "namespace-1",
		FeatureID:        &feature.ID,
		FeatureKey:       &feature.Key,
		UsageAttribution: cust.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   testutils.GetRFC3339Time(t, "2024-08-22T11:25:00Z"),
		})),
		ActiveFrom: lo.ToPtr(testutils.GetRFC3339Time(t, "2024-08-22T11:25:00Z")),
		ActiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2024-08-22T11:30:00Z")),
	})
	assert.NoError(err)
	assert.NotNil(ent1)

	ent2, err := deps.EntitlementConnector.ScheduleEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:        "namespace-1",
		FeatureID:        &feature.ID,
		FeatureKey:       &feature.Key,
		UsageAttribution: cust.GetUsageAttribution(),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodMonth,
			Anchor:   testutils.GetRFC3339Time(t, "2024-08-22T11:25:00Z"),
		})),
		ActiveFrom: lo.ToPtr(testutils.GetRFC3339Time(t, "2024-08-22T11:30:00Z")),
		ActiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2024-08-22T11:35:00Z")),
	})
	assert.NoError(err)
	assert.NotNil(ent2)

	// Lets grant some credit for 500
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-08-22T11:35:18Z"))
	bwRepo, ok := deps.EntitlementRepo.(balanceworker.BalanceWorkerRepository)
	assert.True(ok)

	subjKey, err := cust.UsageAttribution.GetFirstSubjectKey()
	assert.NoError(err)

	affectedEntitlements, err := bwRepo.ListEntitlementsAffectedByIngestEvents(ctx, balanceworker.IngestEventQueryFilter{
		Namespace:    "namespace-1",
		MeterSlugs:   []string{"meter-1"},
		EventSubject: subjKey,
	})
	assert.NoError(err)
	assert.Len(affectedEntitlements, 2)

	entitlements, err := deps.EntitlementConnector.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
		Namespaces:     []string{affectedEntitlements[0].Namespace},
		IDs:            []string{affectedEntitlements[0].EntitlementID},
		IncludeDeleted: true,
	})
	assert.NoError(err)
	assert.Len(entitlements.Items, 1)

	ns := affectedEntitlements[0].Namespace
	entID := affectedEntitlements[0].EntitlementID

	value, err := deps.EntitlementConnector.GetEntitlementValue(ctx, ns, "subject-1", entID, clock.Now())
	assert.NoError(err)

	mappedValues, err := entitlementdriver.MapEntitlementValueToAPI(value)
	assert.NoError(err)
	assert.False(mappedValues.HasAccess)
}
