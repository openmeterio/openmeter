package meteredentitlement_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGetEntitlementBalanceTotalAvailableGrantAmountAfterSnapshot(t *testing.T) {
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
	err = deps.balanceSnapshotService.Save(ctx, models.NamespacedID{
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
		},
	})
	require.NoError(t, err)

	// when: another 100 usage is consumed after the snapshot
	deps.streamingConnector.AddSimpleEvent(meterSlug, 100, snapshotAt.Add(time.Minute))
	entBalance, err := connector.GetEntitlementBalance(ctx, models.NamespacedID{
		Namespace: namespace,
		ID:        ent.ID,
	}, queryAt)
	require.NoError(t, err)

	// then: the period values must remain internally consistent regardless of snapshotting
	require.Equal(t, 300.0, entBalance.UsageInPeriod)
	require.Equal(t, 700.0, entBalance.Balance)
	require.Equal(t, 0.0, entBalance.Overage)
	require.Equal(t, 1000.0, entBalance.TotalAvailableGrantAmount)
}
