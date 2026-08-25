package adapter_test

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	creditadapter "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	db_entitlement "github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type namespaceTestEnv struct {
	db                  *db.Client
	entitlementRepo     entitlement.EntitlementRepo
	grantRepo           grant.Repo
	usageResetRepo      meteredentitlement.UsageResetRepo
	balanceSnapshotRepo balance.SnapshotRepo
}

func newNamespaceTestEnv(t *testing.T) namespaceTestEnv {
	t.Helper()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	dbClient := testDB.EntDriver.Client()

	t.Cleanup(func() {
		_ = dbClient.Close()
		testDB.Close(t)
	})

	return namespaceTestEnv{
		db:                  dbClient,
		entitlementRepo:     entitlementadapter.NewPostgresEntitlementRepo(dbClient),
		grantRepo:           creditadapter.NewPostgresGrantRepo(dbClient),
		usageResetRepo:      entitlementadapter.NewPostgresUsageResetRepo(dbClient),
		balanceSnapshotRepo: creditadapter.NewPostgresBalanceSnapshotRepo(dbClient),
	}
}

func (e namespaceTestEnv) createCustomer(t *testing.T, namespace string) *db.Customer {
	t.Helper()

	key := "customer-" + ulid.Make().String()
	entity, err := e.db.Customer.Create().
		SetNamespace(namespace).
		SetName(key).
		SetKey(key).
		Save(t.Context())
	require.NoError(t, err)

	return entity
}

func (e namespaceTestEnv) createFeature(t *testing.T, namespace string) *db.Feature {
	t.Helper()

	key := "feature-" + ulid.Make().String()
	entity, err := e.db.Feature.Create().
		SetNamespace(namespace).
		SetName(key).
		SetKey(key).
		Save(t.Context())
	require.NoError(t, err)

	return entity
}

func createEntitlementInput(namespace string, customer *db.Customer, feature *db.Feature) entitlement.CreateEntitlementRepoInputs {
	return entitlement.CreateEntitlementRepoInputs{
		Namespace:  namespace,
		FeatureID:  feature.ID,
		FeatureKey: feature.Key,
		UsageAttribution: streaming.CustomerUsageAttribution{
			ID:  customer.ID,
			Key: &customer.Key,
		},
		EntitlementType: entitlement.EntitlementTypeMetered,
	}
}

func TestEntitlementWritesResolveReferencesInNamespace(t *testing.T) {
	// given:
	// - customer, feature, and entitlement rows in target and foreign namespaces
	// when:
	// - repositories receive existing foreign IDs wrapped in the target namespace
	// then:
	// - no entitlement-owned row is persisted with a cross-namespace reference
	env := newNamespaceTestEnv(t)
	targetNamespace := ulid.Make().String()
	foreignNamespace := ulid.Make().String()
	targetCustomer := env.createCustomer(t, targetNamespace)
	targetFeature := env.createFeature(t, targetNamespace)
	foreignCustomer := env.createCustomer(t, foreignNamespace)
	foreignFeature := env.createFeature(t, foreignNamespace)
	mismatchedFeatureKeyInput := createEntitlementInput(targetNamespace, targetCustomer, targetFeature)
	mismatchedFeatureKeyInput.FeatureKey = foreignFeature.Key

	for _, testCase := range []struct {
		name  string
		input entitlement.CreateEntitlementRepoInputs
	}{
		{
			name:  "foreign customer",
			input: createEntitlementInput(targetNamespace, foreignCustomer, targetFeature),
		},
		{
			name:  "foreign feature",
			input: createEntitlementInput(targetNamespace, targetCustomer, foreignFeature),
		},
		{
			name:  "feature key does not match id",
			input: mismatchedFeatureKeyInput,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := env.entitlementRepo.CreateEntitlement(t.Context(), testCase.input)
			require.Error(t, err)
			require.True(t, models.IsGenericNotFoundError(err))
		})
	}

	count, err := env.db.Entitlement.Query().
		Where(db_entitlement.Namespace(targetNamespace)).
		Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, count)

	targetEntitlement, err := env.entitlementRepo.CreateEntitlement(
		t.Context(),
		createEntitlementInput(targetNamespace, targetCustomer, targetFeature),
	)
	require.NoError(t, err)
	require.NotNil(t, targetEntitlement)

	foreignEntitlement, err := env.entitlementRepo.CreateEntitlement(
		t.Context(),
		createEntitlementInput(foreignNamespace, foreignCustomer, foreignFeature),
	)
	require.NoError(t, err)
	require.NotNil(t, foreignEntitlement)

	foreignOwner := models.NamespacedID{
		Namespace: targetNamespace,
		ID:        foreignEntitlement.ID,
	}
	now := time.Now().UTC().Truncate(time.Minute)

	err = env.entitlementRepo.UpsertEntitlementCurrentPeriods(t.Context(), []entitlement.UpsertEntitlementCurrentPeriodElement{
		{
			NamespacedID: foreignOwner,
			CurrentUsagePeriod: timeutil.ClosedPeriod{
				From: now,
				To:   now.AddDate(0, 1, 0),
			},
		},
	})
	var entitlementNotFound *entitlement.NotFoundError
	require.ErrorAs(t, err, &entitlementNotFound)

	foreignEntitlementRow, err := env.db.Entitlement.Get(t.Context(), foreignEntitlement.ID)
	require.NoError(t, err)
	require.Nil(t, foreignEntitlementRow.CurrentUsagePeriodStart)
	require.Nil(t, foreignEntitlementRow.CurrentUsagePeriodEnd)

	_, err = env.grantRepo.CreateGrant(t.Context(), grant.RepoCreateInput{
		Namespace:   foreignOwner.Namespace,
		OwnerID:     foreignOwner.ID,
		Amount:      100,
		EffectiveAt: now,
	})
	require.Error(t, err)
	require.True(t, models.IsGenericNotFoundError(err))

	err = env.usageResetRepo.Save(t.Context(), meteredentitlement.UsageResetUpdate{
		NamespacedModel:     models.NamespacedModel{Namespace: foreignOwner.Namespace},
		EntitlementID:       foreignOwner.ID,
		ResetTime:           now,
		Anchor:              now,
		UsagePeriodInterval: datetime.ISODurationString("P1M"),
	})
	entitlementNotFound = nil
	require.ErrorAs(t, err, &entitlementNotFound)

	err = env.balanceSnapshotRepo.Save(t.Context(), foreignOwner, []balance.Snapshot{
		balance.NewStartingSnapshot(nil, now),
	})
	require.Error(t, err)
	require.True(t, models.IsGenericNotFoundError(err))

	grantCount, err := env.db.Grant.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, grantCount)

	usageResetCount, err := env.db.UsageReset.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, usageResetCount)

	balanceSnapshotCount, err := env.db.BalanceSnapshot.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, balanceSnapshotCount)
}
