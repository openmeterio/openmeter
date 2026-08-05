package meteredentitlement_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	db_balancesnapshot "github.com/openmeterio/openmeter/openmeter/ent/db/balancesnapshot"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestEntitlementBalanceCalculationModesAfterSnapshot(t *testing.T) {
	connector, deps := setupConnector(t)
	defer deps.Teardown()

	ctx := t.Context()
	periodStart := getAnchor(t)
	snapshotAt := periodStart.Add(time.Hour)
	queryAt := snapshotAt.Add(time.Hour)

	feat, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                "feature1",
		Key:                 "feature-1",
		MeterID:             &deps.meterID,
		MeterGroupByFilters: map[string]filter.FilterString{},
	})
	require.NoError(t, err)

	randName := testutils.NameGenerator.Generate()
	cust := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, randName.Key, randName.Name)

	usagePeriod := entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
		Anchor:   periodStart,
		Interval: timeutil.RecurrencePeriodYear,
	})
	currentUsagePeriod, err := usagePeriod.GetValue().GetPeriodAt(queryAt)
	require.NoError(t, err)

	ent, err := deps.entitlementRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
		Namespace:          namespace,
		FeatureID:          feat.ID,
		FeatureKey:         feat.Key,
		UsageAttribution:   cust.GetUsageAttribution(),
		MeasureUsageFrom:   &periodStart,
		EntitlementType:    entitlement.EntitlementTypeMetered,
		IssueAfterReset:    convert.ToPointer(0.0),
		IsSoftLimit:        convert.ToPointer(false),
		UsagePeriod:        &usagePeriod,
		CurrentUsagePeriod: &currentUsagePeriod,
	})
	require.NoError(t, err)

	g, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
		OwnerID:     ent.ID,
		Namespace:   namespace,
		Amount:      1000,
		Priority:    1,
		EffectiveAt: periodStart,
		ExpiresAt:   lo.ToPtr(periodStart.AddDate(1, 0, 0)),
	})
	require.NoError(t, err)

	// given:
	// - 200 usage already consumed from a 1000-unit grant in the current period
	// - a persisted mid-period snapshot carrying the cumulative usage and remaining balance
	deps.streamingConnector.AddSimpleEvent(meterSlug, 200, periodStart.Add(time.Minute))
	err = deps.balanceSnapshotService.SaveComplete(ctx, models.NamespacedID{
		Namespace: namespace,
		ID:        ent.ID,
	}, []balance.Snapshot{
		{
			At:       snapshotAt,
			Balances: balance.Map{g.ID: 800},
			Usage: balance.SnapshottedUsage{
				Since: periodStart,
				Usage: 200,
			},
			UsageSnapshot: &balance.UsageSnapshot{
				Usage:           200,
				TotalGrantUsage: 200,
			},
		},
	})
	require.NoError(t, err)

	// when: another 100 usage is consumed after the snapshot
	deps.streamingConnector.AddSimpleEvent(meterSlug, 100, snapshotAt.Add(time.Minute))
	owner := models.NamespacedID{
		Namespace: namespace,
		ID:        ent.ID,
	}
	value, err := connector.GetValue(ctx, ent, queryAt)
	require.NoError(t, err)
	legacyBalance, ok := value.(*meteredentitlement.MeteredEntitlementValue)
	require.True(t, ok)

	// then: the application-facing calculation retains the legacy,
	// snapshot-relative total during the migration.
	require.Equal(t, 300.0, legacyBalance.UsageInPeriod)
	require.Equal(t, 700.0, legacyBalance.Balance)
	require.Equal(t, 0.0, legacyBalance.Overage)
	require.Equal(t, 800.0, legacyBalance.TotalAvailableGrantAmount)

	completeBalance, err := connector.GetEntitlementBalanceWithCompleteSnapshots(ctx, owner, queryAt)
	require.NoError(t, err)
	require.Equal(t, 300.0, completeBalance.UsageInPeriod)
	require.Equal(t, 700.0, completeBalance.Balance)
	require.Equal(t, 0.0, completeBalance.Overage)
	require.Equal(t, 1000.0, completeBalance.TotalAvailableGrantAmount)

	legacyResult, err := deps.balanceConnector.GetBalanceAt(ctx, owner, queryAt)
	require.NoError(t, err)
	require.Nil(t, legacyResult.Snapshot.UsageSnapshot)

	completeResult, err := deps.balanceConnector.GetBalanceAtWithCompleteSnapshots(ctx, owner, queryAt)
	require.NoError(t, err)
	require.NotNil(t, completeResult.Snapshot.UsageSnapshot)
}

func TestCompleteBalanceSnapshotPersistenceRequiresUsageSnapshot(t *testing.T) {
	_, deps := setupConnector(t)
	defer deps.Teardown()

	ctx := t.Context()
	owner := createBalanceSnapshotOwner(t, deps, getAnchor(t))
	legacySnapshot := balance.Snapshot{
		At:            getAnchor(t),
		Balances:      balance.Map{"grant-1": 800},
		UsageSnapshot: &balance.UsageSnapshot{Usage: 200, TotalGrantUsage: 200},
	}

	err := deps.balanceSnapshotService.Save(ctx, owner, []balance.Snapshot{legacySnapshot})
	require.NoError(t, err)

	savedLegacySnapshot, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, legacySnapshot.At)
	require.NoError(t, err)
	require.Nil(t, savedLegacySnapshot.UsageSnapshot)

	incompleteSnapshot := legacySnapshot.Clone()
	incompleteSnapshot.At = incompleteSnapshot.At.Add(time.Minute)
	incompleteSnapshot.UsageSnapshot = nil
	err = deps.balanceSnapshotService.SaveComplete(ctx, owner, []balance.Snapshot{incompleteSnapshot})
	require.ErrorContains(t, err, "cannot save incomplete balance snapshot")
}

func TestBalanceSnapshotSelectionDuringUsageSnapshotMigration(t *testing.T) {
	_, deps := setupConnector(t)
	defer deps.Teardown()

	ctx := t.Context()
	periodStart := getAnchor(t)
	completeSnapshotAt := periodStart.Add(time.Hour)
	legacySnapshotAt := completeSnapshotAt.Add(time.Hour)
	owner := createBalanceSnapshotOwner(t, deps, periodStart)
	completeSnapshot := balance.Snapshot{
		At:       completeSnapshotAt,
		Balances: balance.Map{"grant-1": 800},
		Usage: balance.SnapshottedUsage{
			Since: periodStart,
			Usage: 200,
		},
		UsageSnapshot: &balance.UsageSnapshot{
			Usage:           200,
			TotalGrantUsage: 200,
		},
	}

	err := deps.balanceSnapshotService.SaveComplete(ctx, owner, []balance.Snapshot{completeSnapshot})
	require.NoError(t, err)

	legacySnapshot := balance.Snapshot{
		At:       legacySnapshotAt,
		Balances: balance.Map{"grant-1": 700},
		Usage: balance.SnapshottedUsage{
			Since: periodStart,
			Usage: 300,
		},
	}
	err = deps.balanceSnapshotService.Save(ctx, owner, []balance.Snapshot{legacySnapshot})
	require.NoError(t, err)

	selectedSnapshot, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, legacySnapshotAt)
	require.NoError(t, err)
	require.Equal(t, legacySnapshot, selectedSnapshot)

	selectedSnapshot, err = deps.balanceSnapshotService.GetLatestValidCompleteAt(ctx, owner, legacySnapshotAt)
	require.NoError(t, err)
	require.Equal(t, completeSnapshot, selectedSnapshot)

	_, err = deps.dbClient.BalanceSnapshot.Delete().
		Where(
			db_balancesnapshot.Namespace(owner.Namespace),
			db_balancesnapshot.OwnerID(owner.ID),
			db_balancesnapshot.UsageSnapshotNotNil(),
		).
		Exec(ctx)
	require.NoError(t, err)

	selectedSnapshot, err = deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, legacySnapshotAt)
	require.NoError(t, err)
	require.Equal(t, legacySnapshot, selectedSnapshot)

	_, err = deps.balanceSnapshotService.GetLatestValidCompleteAt(ctx, owner, legacySnapshotAt)
	var noSavedSnapshot *balance.NoSavedBalanceForOwnerError
	require.ErrorAs(t, err, &noSavedSnapshot)
}

func TestBalanceSnapshotVersionsAreIndependent(t *testing.T) {
	_, deps := setupConnector(t)
	defer deps.Teardown()

	ctx := t.Context()
	periodStart := getAnchor(t)
	snapshotAt := periodStart.Add(time.Hour)
	queryAt := snapshotAt.Add(time.Hour)
	owner := createBalanceSnapshotOwner(t, deps, periodStart)
	snapshotA := balance.Snapshot{
		At:       snapshotAt,
		Balances: balance.Map{"grant-1": 800},
		Usage: balance.SnapshottedUsage{
			Since: periodStart,
			Usage: 200,
		},
		UsageSnapshot: &balance.UsageSnapshot{
			Usage:           200,
			TotalGrantUsage: 200,
		},
	}
	snapshotB := balance.Snapshot{
		At:       snapshotAt.Add(10 * time.Minute),
		Balances: balance.Map{"grant-1": 700},
		Usage: balance.SnapshottedUsage{
			Since: periodStart,
			Usage: 300,
		},
		UsageSnapshot: &balance.UsageSnapshot{
			Usage:           300,
			TotalGrantUsage: 300,
		},
	}
	snapshotC := balance.Snapshot{
		At:       snapshotAt.Add(20 * time.Minute),
		Balances: balance.Map{"grant-1": 600},
		Usage: balance.SnapshottedUsage{
			Since: periodStart,
			Usage: 400,
		},
		UsageSnapshot: &balance.UsageSnapshot{
			Usage:           400,
			TotalGrantUsage: 400,
		},
	}

	err := deps.balanceSnapshotService.SaveComplete(ctx, owner, []balance.Snapshot{snapshotA, snapshotB, snapshotC})
	require.NoError(t, err)

	_, err = deps.dbClient.BalanceSnapshot.Delete().
		Where(
			db_balancesnapshot.Namespace(owner.Namespace),
			db_balancesnapshot.OwnerID(owner.ID),
			db_balancesnapshot.AtEQ(snapshotB.At),
		).
		Exec(ctx)
	require.NoError(t, err)

	selectedSnapshot, err := deps.balanceSnapshotService.GetLatestValidCompleteAt(ctx, owner, queryAt)
	require.NoError(t, err)
	require.Equal(t, snapshotC, selectedSnapshot)

	_, err = deps.dbClient.BalanceSnapshot.Delete().
		Where(
			db_balancesnapshot.Namespace(owner.Namespace),
			db_balancesnapshot.OwnerID(owner.ID),
			db_balancesnapshot.AtEQ(snapshotA.At),
		).
		Exec(ctx)
	require.NoError(t, err)

	selectedSnapshot, err = deps.balanceSnapshotService.GetLatestValidCompleteAt(ctx, owner, queryAt)
	require.NoError(t, err)
	require.Equal(t, snapshotC, selectedSnapshot)

	err = deps.balanceSnapshotService.SaveComplete(ctx, owner, []balance.Snapshot{snapshotB})
	require.NoError(t, err)

	selectedSnapshot, err = deps.balanceSnapshotService.GetLatestValidCompleteAt(ctx, owner, queryAt)
	require.NoError(t, err)
	require.Equal(t, snapshotC, selectedSnapshot)
}

func TestCompleteCalculationCreatesUsageSnapshotWithoutChangingLegacySelection(t *testing.T) {
	connector, deps := setupConnector(t)
	defer deps.Teardown()

	ctx := t.Context()
	periodStart := getAnchor(t)
	breakpointAt := periodStart.Add(time.Hour)
	legacySnapshotAt := breakpointAt.Add(time.Hour)
	queryAt := periodStart.AddDate(0, 0, 10)
	owner := createBalanceSnapshotOwner(t, deps, periodStart)

	firstGrant, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
		OwnerID:     owner.ID,
		Namespace:   owner.Namespace,
		Amount:      1000,
		Priority:    1,
		EffectiveAt: periodStart,
		ExpiresAt:   lo.ToPtr(periodStart.AddDate(1, 0, 0)),
	})
	require.NoError(t, err)
	secondGrant, err := deps.grantRepo.CreateGrant(ctx, grant.RepoCreateInput{
		OwnerID:     owner.ID,
		Namespace:   owner.Namespace,
		Amount:      100,
		Priority:    2,
		EffectiveAt: breakpointAt,
		ExpiresAt:   lo.ToPtr(periodStart.AddDate(1, 0, 0)),
	})
	require.NoError(t, err)

	deps.streamingConnector.AddSimpleEvent(meterSlug, 200, periodStart.Add(time.Minute))
	deps.streamingConnector.AddSimpleEvent(meterSlug, 100, legacySnapshotAt.Add(time.Minute))

	legacySnapshot := balance.Snapshot{
		At: legacySnapshotAt,
		Balances: balance.Map{
			firstGrant.ID:  700,
			secondGrant.ID: 100,
		},
		Usage: balance.SnapshottedUsage{
			Since: periodStart,
			Usage: 300,
		},
	}
	require.NoError(t, deps.balanceSnapshotService.Save(ctx, owner, []balance.Snapshot{legacySnapshot}))

	_, err = connector.GetEntitlementBalanceWithCompleteSnapshots(ctx, owner, queryAt)
	require.NoError(t, err)

	selectedLegacySnapshot, err := deps.balanceSnapshotService.GetLatestValidAt(ctx, owner, queryAt)
	require.NoError(t, err)
	require.Equal(t, legacySnapshot, selectedLegacySnapshot)

	completeSnapshot, err := deps.balanceSnapshotService.GetLatestValidCompleteAt(ctx, owner, queryAt)
	require.NoError(t, err)
	require.Equal(t, breakpointAt, completeSnapshot.At)
	require.Equal(t, balance.UsageSnapshot{
		Usage:           200,
		TotalGrantUsage: 200,
	}, *completeSnapshot.UsageSnapshot)
}

func createBalanceSnapshotOwner(t *testing.T, deps *dependencies, at time.Time) models.NamespacedID {
	ctx := t.Context()
	featureName := testutils.NameGenerator.Generate()
	feat, err := deps.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                featureName.Name,
		Key:                 featureName.Key,
		MeterID:             &deps.meterID,
		MeterGroupByFilters: map[string]filter.FilterString{},
	})
	require.NoError(t, err)

	customerName := testutils.NameGenerator.Generate()
	cust := createCustomerAndSubject(
		t,
		deps.subjectService,
		deps.customerService,
		namespace,
		customerName.Key,
		customerName.Name,
	)
	usagePeriod := entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
		Anchor:   at,
		Interval: timeutil.RecurrencePeriodYear,
	})
	currentUsagePeriod, err := usagePeriod.GetValue().GetPeriodAt(at)
	require.NoError(t, err)

	ent, err := deps.entitlementRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
		Namespace:          namespace,
		FeatureID:          feat.ID,
		FeatureKey:         feat.Key,
		UsageAttribution:   cust.GetUsageAttribution(),
		MeasureUsageFrom:   &at,
		EntitlementType:    entitlement.EntitlementTypeMetered,
		IssueAfterReset:    convert.ToPointer(0.0),
		IsSoftLimit:        convert.ToPointer(false),
		UsagePeriod:        &usagePeriod,
		CurrentUsagePeriod: &currentUsagePeriod,
	})
	require.NoError(t, err)

	return models.NamespacedID{
		Namespace: namespace,
		ID:        ent.ID,
	}
}
