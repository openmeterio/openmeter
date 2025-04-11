package adapter_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	featureadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var m sync.Mutex

type deps struct {
	entRepo     entitlement.EntitlementRepo
	featureRepo feature.FeatureRepo
}

func setup(t *testing.T) (deps deps, cleanup func()) {
	t.Helper()

	// create isolated pg db for tests
	testdb := testutils.InitPostgresDB(t)
	logger := testutils.NewLogger(t)

	logger.Info("testdb.URL", "testdb.URL", testdb.URL)

	dbClient := testdb.EntDriver.Client().Debug()
	pgDriver := testdb.PGDriver
	entDriver := testdb.EntDriver

	cleanup = func() {
		dbClient.Close()
		entDriver.Close()
		pgDriver.Close()
	}

	deps.entRepo = adapter.NewPostgresEntitlementRepo(dbClient)
	deps.featureRepo = featureadapter.NewPostgresFeatureRepo(dbClient, logger)

	m.Lock()
	defer m.Unlock()
	// migrate db via ent schema upsert
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return deps, cleanup
}

func TestUpsertEntitlementCurrentPeriods(t *testing.T) {
	ns := "ns1"
	featureKey := "feature1"

	t.Run("Should upsert entitlement current periods but no other fields", func(t *testing.T) {
		ctx := context.Background()
		repo, cleanup := setup(t)
		defer cleanup()

		// Let's create an example feature
		feature, err := repo.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: ns,
			Key:       featureKey,
			Name:      "Feature 1",
		})
		require.NoError(t, err)

		// First, let's create 3 entitlements
		ent1, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        ns,
			FeatureID:        feature.ID,
			FeatureKey:       featureKey,
			SubjectKey:       "subject1",
			EntitlementType:  entitlement.EntitlementTypeMetered,
			MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			UsagePeriod: &entitlement.UsagePeriod{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			CurrentUsagePeriod: &timeutil.ClosedPeriod{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, ent1)

		ent2, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        ns,
			FeatureID:        feature.ID,
			FeatureKey:       featureKey,
			SubjectKey:       "subject2",
			EntitlementType:  entitlement.EntitlementTypeMetered,
			MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			UsagePeriod: &entitlement.UsagePeriod{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			CurrentUsagePeriod: nil, // notice we don't have a current period set here
		})
		require.NoError(t, err)
		require.NotNil(t, ent2)

		ent3, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        ns,
			FeatureID:        feature.ID,
			FeatureKey:       featureKey,
			SubjectKey:       "subject3",
			EntitlementType:  entitlement.EntitlementTypeMetered,
			MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			UsagePeriod: &entitlement.UsagePeriod{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			CurrentUsagePeriod: &timeutil.ClosedPeriod{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, ent3)

		ent1NewPeriod := timeutil.ClosedPeriod{
			From: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
			To:   time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
		}

		ent2NewPeriod := timeutil.ClosedPeriod{
			From: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
			To:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		}

		// Then let's upsert two of them
		err = repo.entRepo.UpsertEntitlementCurrentPeriods(ctx, []entitlement.UpsertEntitlementCurrentPeriodElement{
			{
				NamespacedID: models.NamespacedID{
					ID:        ent1.ID,
					Namespace: ent1.Namespace,
				},
				CurrentUsagePeriod: ent1NewPeriod,
			},
			{
				NamespacedID: models.NamespacedID{
					ID:        ent2.ID,
					Namespace: ent2.Namespace,
				},
				CurrentUsagePeriod: ent2NewPeriod,
			},
		})
		require.NoError(t, err)

		// Let's check we still only have 3 entitlements
		entitlements, err := repo.entRepo.ListEntitlements(ctx, entitlement.ListEntitlementsParams{})
		require.NoError(t, err)
		require.Equal(t, 3, len(entitlements.Items))

		// To avoid mapping to calculated values, we need to use ListActiveEntitlementsWithExpiredUsagePeriod
		ents, err := repo.entRepo.ListActiveEntitlementsWithExpiredUsagePeriod(ctx, entitlement.ListExpiredEntitlementsParams{
			Namespaces:    []string{ns},
			Highwatermark: time.Date(2030, 4, 1, 0, 0, 0, 0, time.UTC),
		})
		require.NoError(t, err)
		require.Equal(t, 3, len(ents))

		// Let's check that their current periods are updated and no other fields are touched
		ent1Updated, ok := lo.Find(ents, func(e entitlement.Entitlement) bool {
			return e.ID == ent1.ID
		})
		require.True(t, ok)
		require.NotNil(t, ent1Updated)
		require.Equal(t, ent1.ID, ent1Updated.ID)
		require.NotNil(t, ent1Updated.CurrentUsagePeriod)
		require.Equal(t, ent1NewPeriod, *ent1Updated.CurrentUsagePeriod)
		require.Equal(t, ent1.FeatureID, ent1Updated.FeatureID)
		require.Equal(t, ent1.FeatureKey, ent1Updated.FeatureKey)
		require.Equal(t, ent1.SubjectKey, ent1Updated.SubjectKey)
		require.Equal(t, ent1.EntitlementType, ent1Updated.EntitlementType)
		require.Equal(t, ent1.MeasureUsageFrom.UTC(), ent1Updated.MeasureUsageFrom.UTC())
		require.Equal(t, ent1.UsagePeriod, ent1Updated.UsagePeriod)

		ent2Updated, ok := lo.Find(ents, func(e entitlement.Entitlement) bool {
			return e.ID == ent2.ID
		})
		require.True(t, ok)
		require.NotNil(t, ent2Updated)
		require.Equal(t, ent2.ID, ent2Updated.ID)
		require.NotNil(t, ent2Updated.CurrentUsagePeriod)
		require.Equal(t, ent2NewPeriod, *ent2Updated.CurrentUsagePeriod)
		require.Equal(t, ent2.FeatureID, ent2Updated.FeatureID)
		require.Equal(t, ent2.FeatureKey, ent2Updated.FeatureKey)
		require.Equal(t, ent2.SubjectKey, ent2Updated.SubjectKey)
		require.Equal(t, ent2.EntitlementType, ent2Updated.EntitlementType)
		require.Equal(t, ent2.MeasureUsageFrom.UTC(), ent2Updated.MeasureUsageFrom.UTC())
		require.Equal(t, ent2.UsagePeriod, ent2Updated.UsagePeriod)

		// Let's check that the other one is not touched
		ent3Updated, ok := lo.Find(entitlements.Items, func(e entitlement.Entitlement) bool {
			return e.ID == ent3.ID
		})
		require.True(t, ok)
		require.NotNil(t, ent3Updated)
		require.Equal(t, ent3.ID, ent3Updated.ID)
		require.Equal(t, ent3.CurrentUsagePeriod, ent3Updated.CurrentUsagePeriod)
		require.Equal(t, ent3.FeatureID, ent3Updated.FeatureID)
		require.Equal(t, ent3.FeatureKey, ent3Updated.FeatureKey)
		require.Equal(t, ent3.SubjectKey, ent3Updated.SubjectKey)
		require.Equal(t, ent3.EntitlementType, ent3Updated.EntitlementType)
		require.Equal(t, ent3.MeasureUsageFrom.UTC(), ent3Updated.MeasureUsageFrom.UTC())
		require.Equal(t, ent3.UsagePeriod, ent3Updated.UsagePeriod)
	})
}

func TestListActiveEntitlementsWithExpiredUsagePeriod(t *testing.T) {
	ns := "ns1"
	featureKey := "feature1"

	t.Run("Should return entitlements with expired usage period", func(t *testing.T) {
		ctx := context.Background()
		repo, cleanup := setup(t)
		defer cleanup()

		// Let's create an example feature
		feature, err := repo.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: ns,
			Key:       featureKey,
			Name:      "Feature 1",
		})
		require.NoError(t, err)

		// Let's set the current time
		now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		clock.SetTime(now)
		defer clock.ResetTime()

		// Let's create two entitlements, one with expired usage period and one with no expired usage period
		ent1, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        ns,
			FeatureID:        feature.ID,
			FeatureKey:       featureKey,
			SubjectKey:       "subject1",
			EntitlementType:  entitlement.EntitlementTypeMetered,
			MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			UsagePeriod: &entitlement.UsagePeriod{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			CurrentUsagePeriod: &timeutil.ClosedPeriod{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, ent1)

		ent2, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
			Namespace:        ns,
			FeatureID:        feature.ID,
			FeatureKey:       featureKey,
			SubjectKey:       "subject2",
			EntitlementType:  entitlement.EntitlementTypeMetered,
			MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			UsagePeriod: &entitlement.UsagePeriod{
				Interval: timeutil.RecurrencePeriodMonth,
				Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			CurrentUsagePeriod: &timeutil.ClosedPeriod{
				From: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC),
			},
		})
		require.NoError(t, err)
		require.NotNil(t, ent2)

		now = time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
		clock.SetTime(now)
		defer clock.ResetTime()

		// Let's check that the entitlement with expired usage period is returned
		ents, err := repo.entRepo.ListActiveEntitlementsWithExpiredUsagePeriod(ctx, entitlement.ListExpiredEntitlementsParams{
			Namespaces:    []string{ns},
			Highwatermark: now,
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(ents))
		require.Equal(t, ent1.ID, ents[0].ID)
	})

	t.Run("Should return entitlements with cursor and limit", func(t *testing.T) {
		ctx := context.Background()
		repo, cleanup := setup(t)
		defer cleanup()

		// Let's create an example feature
		feature, err := repo.featureRepo.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: ns,
			Key:       featureKey,
			Name:      "Feature 1",
		})
		require.NoError(t, err)

		var ents []entitlement.Entitlement

		// Let's create 10 entitlements
		for i := 0; i < 10; i++ {
			ent, err := repo.entRepo.CreateEntitlement(ctx, entitlement.CreateEntitlementRepoInputs{
				Namespace:        ns,
				FeatureID:        feature.ID,
				FeatureKey:       featureKey,
				SubjectKey:       fmt.Sprintf("subject%d", i),
				EntitlementType:  entitlement.EntitlementTypeMetered,
				MeasureUsageFrom: lo.ToPtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
				UsagePeriod: &entitlement.UsagePeriod{
					Interval: timeutil.RecurrencePeriodMonth,
					Anchor:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				CurrentUsagePeriod: &timeutil.ClosedPeriod{
					From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
				},
			})
			require.NoError(t, err)
			require.NotNil(t, ent)
			ents = append(ents, *ent)
		}

		// Let's check that the entitlements are returned
		resetableEnts, err := repo.entRepo.ListActiveEntitlementsWithExpiredUsagePeriod(ctx, entitlement.ListExpiredEntitlementsParams{
			Namespaces:    []string{ns},
			Highwatermark: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
			Limit:         5,
		})
		require.NoError(t, err)
		require.Equal(t, 5, len(resetableEnts))
		require.Equal(t, ents[0].ID, resetableEnts[0].ID)
		require.Equal(t, ents[1].ID, resetableEnts[1].ID)
		require.Equal(t, ents[2].ID, resetableEnts[2].ID)
		require.Equal(t, ents[3].ID, resetableEnts[3].ID)
		require.Equal(t, ents[4].ID, resetableEnts[4].ID)

		// Let's query the next 5 entitlements
		next5Ents, err := repo.entRepo.ListActiveEntitlementsWithExpiredUsagePeriod(ctx, entitlement.ListExpiredEntitlementsParams{
			Namespaces:    []string{ns},
			Highwatermark: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
			Cursor:        resetableEnts[4].ID,
		})
		require.NoError(t, err)
		require.Equal(t, 5, len(next5Ents))
		require.Equal(t, ents[5].ID, next5Ents[0].ID)
		require.Equal(t, ents[6].ID, next5Ents[1].ID)
		require.Equal(t, ents[7].ID, next5Ents[2].ID)
		require.Equal(t, ents[8].ID, next5Ents[3].ID)
		require.Equal(t, ents[9].ID, next5Ents[4].ID)
	})
}
