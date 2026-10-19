package meteredentitlement_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type blockingQueryStreamingConnector struct {
	streaming.Connector
	queryStarted  chan struct{}
	continueQuery chan struct{}
	startOnce     sync.Once
	continueOnce  sync.Once
}

func (c *blockingQueryStreamingConnector) QueryMeter(ctx context.Context, namespace string, m meter.Meter, params streaming.QueryParams) ([]meter.MeterQueryRow, error) {
	c.startOnce.Do(func() {
		close(c.queryStarted)
	})

	select {
	case <-c.continueQuery:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return c.Connector.QueryMeter(ctx, namespace, m, params)
}

func (c *blockingQueryStreamingConnector) continueQuerying() {
	c.continueOnce.Do(func() {
		close(c.continueQuery)
	})
}

func TestBalanceCalculationDoesNotSaveSnapshotAfterConcurrentGrant(t *testing.T) {
	var streamingConnector *blockingQueryStreamingConnector
	_, deps := setupConnectorWithStreaming(t, func(connector streaming.Connector) streaming.Connector {
		streamingConnector = &blockingQueryStreamingConnector{
			Connector:     connector,
			queryStarted:  make(chan struct{}),
			continueQuery: make(chan struct{}),
		}

		return streamingConnector
	})
	defer deps.Teardown()
	defer streamingConnector.continueQuerying()

	start := getAnchor(t)
	queryAt := start.AddDate(0, 0, 9)
	clock.FreezeTime(queryAt)
	defer clock.UnFreeze()

	feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                "snapshot invalidation",
		Key:                 "snapshot-invalidation",
		MeterID:             &deps.meterID,
		MeterGroupByFilters: map[string]filter.FilterString{},
	})
	require.NoError(t, err)

	name := testutils.NameGenerator.Generate()
	customer := createCustomerAndSubject(t, deps.subjectService, deps.customerService, namespace, name.Key, name.Name)
	usagePeriod := entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
		Anchor:   start,
		Interval: timeutil.RecurrencePeriodDaily,
	})
	currentUsagePeriod, err := usagePeriod.GetValue().GetPeriodAt(queryAt)
	require.NoError(t, err)

	ent, err := deps.entitlementRepo.CreateEntitlement(t.Context(), entitlement.CreateEntitlementRepoInputs{
		Namespace:          namespace,
		FeatureID:          feat.ID,
		FeatureKey:         feat.Key,
		UsageAttribution:   customer.GetUsageAttribution(),
		EntitlementType:    entitlement.EntitlementTypeMetered,
		MeasureUsageFrom:   &start,
		IssueAfterReset:    lo.ToPtr(0.0),
		IsSoftLimit:        lo.ToPtr(false),
		UsagePeriod:        &usagePeriod,
		CurrentUsagePeriod: &currentUsagePeriod,
	})
	require.NoError(t, err)

	owner := models.NamespacedID{Namespace: namespace, ID: ent.ID}
	_, err = deps.grantRepo.CreateGrant(t.Context(), grant.RepoCreateInput{
		Namespace:   namespace,
		OwnerID:     owner.ID,
		Amount:      100,
		EffectiveAt: start,
	})
	require.NoError(t, err)
	deps.streamingConnector.AddSimpleEvent(meterSlug, 1, start.Add(time.Minute))

	result := make(chan error, 1)
	go func() {
		_, err := deps.balanceConnector.GetBalanceAt(t.Context(), owner, queryAt)
		result <- err
	}()

	select {
	case <-streamingConnector.queryStarted:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "balance calculation did not query usage")
	}

	_, err = deps.creditConnector.CreateGrant(t.Context(), owner, credit.CreateGrantInput{
		Amount:      50,
		EffectiveAt: queryAt,
	})
	require.NoError(t, err)
	streamingConnector.continueQuerying()

	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		require.FailNow(t, "balance calculation did not finish")
	}

	version, err := deps.balanceSnapshotService.GetInvalidationVersion(t.Context(), owner)
	require.NoError(t, err)
	require.Equal(t, balance.SnapshotInvalidationVersion(1), version)

	_, err = deps.balanceSnapshotService.GetLatestValidAt(t.Context(), owner, queryAt)
	var noSnapshot *balance.NoSavedBalanceForOwnerError
	require.ErrorAs(t, err, &noSnapshot)
}
