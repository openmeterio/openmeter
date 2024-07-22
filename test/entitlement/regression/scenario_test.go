package framework_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/internal/entitlement/metered"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

func TestGrantExpiringAtReset(t *testing.T) {
	defer clock.ResetTime()
	deps := setupDependencies(t)
	defer deps.Close()
	ctx := context.Background()
	assert := assert.New(t)

	// Let's create a feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:30:21Z"))
	feature, err := deps.FeatureConnector.CreateFeature(ctx, productcatalog.CreateFeatureInputs{
		Name:      "feature-1",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feature)

	// Let's create a new entitlement for the feature

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:35:21Z"))
	entitlement, err := deps.EntitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:       "namespace-1",
		FeatureID:       &feature.ID,
		FeatureKey:      &feature.Key,
		SubjectKey:      "subject-1",
		EntitlementType: entitlement.EntitlementTypeMetered,
		UsagePeriod: &entitlement.UsagePeriod{
			Interval: recurrence.RecurrencePeriodDaily,
			Anchor:   testutils.GetRFC3339Time(t, "2024-06-28T14:48:00Z"),
		},
	})
	assert.NoError(err)
	assert.NotNil(entitlement)

	// Let's grant some credit

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:35:24Z"))
	grant1, err := deps.GrantConnector.CreateGrant(ctx,
		credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        credit.GrantOwner(entitlement.ID),
		},
		credit.CreateGrantInput{
			Amount:      10,
			Priority:    5,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-06-28T14:35:00Z"),
			Expiration: credit.ExpirationPeriod{
				Count:    1,
				Duration: credit.ExpirationPeriodDurationYear,
			},
		})
	assert.NoError(err)
	assert.NotNil(grant1)

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-28T14:36:33Z"))
	grant2, err := deps.GrantConnector.CreateGrant(ctx,
		credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        credit.GrantOwner(entitlement.ID),
		},
		credit.CreateGrantInput{
			Amount:      20,
			Priority:    3,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-06-28T14:36:00Z"),
			Expiration: credit.ExpirationPeriod{
				Count:    1,
				Duration: credit.ExpirationPeriodDurationDay,
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
		testutils.GetRFC3339Time(t, "2024-06-28T14:30:41Z"))
	assert.NoError(err)
	assert.NotNil(currentBalance)
	assert.Equal(10.0, currentBalance.Balance)

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-30T15:35:54Z"))
	grant3, err := deps.GrantConnector.CreateGrant(ctx,
		credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        credit.GrantOwner(entitlement.ID),
		},
		credit.CreateGrantInput{
			Amount:      100,
			Priority:    1,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-06-28T15:39:00Z"),
			Expiration: credit.ExpirationPeriod{
				Count:    1,
				Duration: credit.ExpirationPeriodDurationYear,
			},
		})
	assert.NoError(err)
	assert.NotNil(grant3)

	// There should be a snapshot created
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-06-30T15:37:18Z"))
	reset, err := deps.MeteredEntitlementConnector.ResetEntitlementUsage(ctx,
		models.NamespacedID{
			Namespace: "namespace-1",
			ID:        entitlement.ID,
		},
		meteredentitlement.ResetEntitlementUsageParams{
			At:           testutils.GetRFC3339Time(t, "2024-06-29T14:36:00Z"),
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
	feature, err := deps.FeatureConnector.CreateFeature(ctx, productcatalog.CreateFeatureInputs{
		Name:      "feature-1",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feature)

	// Let's create a new entitlement for the feature

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-02T09:41:14Z"))
	entitlement, err := deps.EntitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:       "namespace-1",
		FeatureID:       &feature.ID,
		FeatureKey:      &feature.Key,
		SubjectKey:      "subject-1",
		EntitlementType: entitlement.EntitlementTypeMetered,
		UsagePeriod: &entitlement.UsagePeriod{
			Interval: recurrence.RecurrencePeriodDaily,
			Anchor:   testutils.GetRFC3339Time(t, "2024-07-02T09:41:00Z"),
		},
	})
	assert.NoError(err)
	assert.NotNil(entitlement)

	// Let's grant some credit

	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-02T09:43:04Z"))
	grant1, err := deps.GrantConnector.CreateGrant(ctx,
		credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        credit.GrantOwner(entitlement.ID),
		},
		credit.CreateGrantInput{
			Amount:           20,
			ResetMaxRollover: 20,
			Priority:         1,
			EffectiveAt:      testutils.GetRFC3339Time(t, "2024-07-02T09:43:00Z"),
			Expiration: credit.ExpirationPeriod{
				Count:    1,
				Duration: credit.ExpirationPeriodDurationWeek,
			},
			Recurrence: &recurrence.Recurrence{
				Interval: recurrence.RecurrencePeriodDaily,
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

	// Validate snapshot exists
	snapshot, err := deps.BalanceSnapshotRepo.GetLatestValidAt(ctx, credit.NamespacedGrantOwner{
		Namespace: "namespace-1",
		ID:        credit.GrantOwner(entitlement.ID),
	}, testutils.GetRFC3339Time(t, "2024-07-10T07:33:06Z"))
	assert.NoError(err)
	assert.NotNil(snapshot)
	assert.NotEqual(resetCommand.At, snapshot.At)
	assert.Greater(snapshot.At.Unix(), resetCommand.At.Unix())
	assert.Equal(currentBalance.Balance, snapshot.Balances.Balance()) // because there's nothing it's just 0

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
	feature, err := deps.FeatureConnector.CreateFeature(ctx, productcatalog.CreateFeatureInputs{
		Name:      "feature-1",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feature)

	// Let's create a new entitlement for the feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T11:20:28Z"))
	entitlement, err := deps.EntitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:       "namespace-1",
		FeatureID:       &feature.ID,
		FeatureKey:      &feature.Key,
		SubjectKey:      "subject-1",
		IssueAfterReset: convert.ToPointer(500.0),
		EntitlementType: entitlement.EntitlementTypeMetered,
		UsagePeriod: &entitlement.UsagePeriod{
			Interval: recurrence.RecurrencePeriodMonth,
			Anchor:   testutils.GetRFC3339Time(t, "2024-07-01T00:00:00Z"),
		},
	})
	assert.NoError(err)
	assert.NotNil(entitlement)

	// Let's retrieve the grant so we can reference it
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T12:20:28Z"))
	grants, err := deps.GrantConnector.ListGrants(ctx, credit.ListGrantsParams{
		Namespace:      "namespace-1",
		IncludeDeleted: true,
		Offset:         0,
		Limit:          100,
		OrderBy:        credit.GrantOrderByCreatedAt,
	})
	assert.NoError(err)
	assert.Len(grants, 1)

	grant1 := &grants[0]

	// Let's create another grant
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T12:09:40Z"))
	grant2, err := deps.GrantConnector.CreateGrant(ctx,
		credit.NamespacedGrantOwner{
			Namespace: "namespace-1",
			ID:        credit.GrantOwner(entitlement.ID),
		},
		credit.CreateGrantInput{
			Amount:      10000,
			Priority:    1,
			EffectiveAt: testutils.GetRFC3339Time(t, "2024-07-09T12:09:00Z"),
			Expiration: credit.ExpirationPeriod{
				Count:    1,
				Duration: credit.ExpirationPeriodDurationWeek,
			},
		})
	assert.NoError(err)
	assert.NotNil(grant2)

	// Lets create a snapshot
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T13:09:05Z"))
	err = deps.BalanceSnapshotRepo.Save(ctx, credit.NamespacedGrantOwner{
		Namespace: "namespace-1",
		ID:        credit.GrantOwner(entitlement.ID),
	}, []credit.GrantBalanceSnapshot{
		{
			At:      testutils.GetRFC3339Time(t, "2024-07-09T13:09:00Z"),
			Overage: 0.0,
			Balances: credit.GrantBalanceMap{
				grant1.ID: 488.0,
				grant2.ID: 10000.0,
			},
		},
	})
	assert.NoError(err)

	// Hack: this is in the future, but at least it won't return an error
	deps.Streaming.AddSimpleEvent("meter-1", 1, testutils.GetRFC3339Time(t, "2099-06-28T14:36:00Z"))

	// Lets void the grant
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T14:54:04Z"))
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
	feature, err := deps.FeatureConnector.CreateFeature(ctx, productcatalog.CreateFeatureInputs{
		Name:      "feature-1",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feature)

	// Let's create a new entitlement for the feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T11:20:28Z"))
	ent, err := deps.EntitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:       "namespace-1",
		FeatureID:       &feature.ID,
		FeatureKey:      &feature.Key,
		SubjectKey:      "subject-1",
		IssueAfterReset: convert.ToPointer(500.0),
		EntitlementType: entitlement.EntitlementTypeMetered,
		UsagePeriod: &entitlement.UsagePeriod{
			Interval: recurrence.RecurrencePeriodMonth,
			Anchor:   testutils.GetRFC3339Time(t, "2024-07-01T00:00:00Z"),
		},
	})
	assert.NoError(err)
	assert.NotNil(ent)

	// Let's archive the feature
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T12:20:28Z"))
	err = deps.FeatureConnector.ArchiveFeature(ctx, models.NamespacedID{
		Namespace: "namespace-1",
		ID:        feature.ID,
	})
	assert.NoError(err)

	// Let's create a new feature with the same key
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T13:20:28Z"))
	feature2, err := deps.FeatureConnector.CreateFeature(ctx, productcatalog.CreateFeatureInputs{
		Name:      "feature-1-2",
		Key:       "feature-1",
		Namespace: "namespace-1",
		MeterSlug: convert.ToPointer("meter-1"),
	})
	assert.NoError(err)
	assert.NotNil(feature2)

	// Let's create a new entitlement for feature2 for subject-2
	clock.SetTime(testutils.GetRFC3339Time(t, "2024-07-09T14:20:28Z"))
	ent2, err := deps.EntitlementConnector.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:       "namespace-1",
		FeatureID:       &feature2.ID,
		FeatureKey:      &feature2.Key,
		SubjectKey:      "subject-2",
		IssueAfterReset: convert.ToPointer(500.0),
		EntitlementType: entitlement.EntitlementTypeMetered,
		UsagePeriod: &entitlement.UsagePeriod{
			Interval: recurrence.RecurrencePeriodMonth,
			Anchor:   testutils.GetRFC3339Time(t, "2024-07-01T00:00:00Z"),
		},
	})
	assert.NoError(err)
	assert.NotNil(ent2)
}
