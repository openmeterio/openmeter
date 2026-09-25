package adapter_test

import (
	"slices"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	creditadapter "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func TestListGrantsBreaksSortTiesByID(t *testing.T) {
	// given:
	// - grants of one entitlement that share the same effective time
	// when:
	// - they are listed one per page, sorted by effective time
	// then:
	// - every grant appears exactly once, with ties ordered by ID in the requested direction
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	dbClient := testDB.EntDriver.Client()
	t.Cleanup(func() {
		_ = dbClient.Close()
		testDB.Close(t)
	})

	namespace := ulid.Make().String()
	key := "grant-order-" + ulid.Make().String()

	customer, err := dbClient.Customer.Create().SetNamespace(namespace).SetName(key).SetKey(key).Save(t.Context())
	require.NoError(t, err)
	feature, err := dbClient.Feature.Create().SetNamespace(namespace).SetName(key).SetKey(key).Save(t.Context())
	require.NoError(t, err)

	ent, err := entitlementadapter.NewPostgresEntitlementRepo(dbClient).CreateEntitlement(t.Context(), entitlement.CreateEntitlementRepoInputs{
		Namespace:  namespace,
		FeatureID:  feature.ID,
		FeatureKey: feature.Key,
		UsageAttribution: streaming.CustomerUsageAttribution{
			ID:  customer.ID,
			Key: &customer.Key,
		},
		EntitlementType: entitlement.EntitlementTypeMetered,
	})
	require.NoError(t, err)

	repo := creditadapter.NewPostgresGrantRepo(dbClient)
	effectiveAt := time.Now().UTC().Truncate(time.Minute)

	var ids []string
	for range 4 {
		g, err := repo.CreateGrant(t.Context(), grant.RepoCreateInput{
			Namespace:   namespace,
			OwnerID:     ent.ID,
			Amount:      100,
			EffectiveAt: effectiveAt,
		})
		require.NoError(t, err)
		ids = append(ids, g.ID)
	}
	slices.Sort(ids)

	listPaged := func(t *testing.T, order sortx.Order) []string {
		var listed []string
		for page := 1; page <= len(ids); page++ {
			res, err := repo.ListGrants(t.Context(), grant.ListParams{
				Namespace: namespace,
				OwnerID:   &ent.ID,
				OrderBy:   grant.OrderByEffectiveAt,
				Order:     order,
				Page:      pagination.NewPage(page, 1),
			})
			require.NoError(t, err)
			require.Len(t, res.Items, 1)
			listed = append(listed, res.Items[0].ID)
		}
		return listed
	}

	require.Equal(t, ids, listPaged(t, sortx.OrderAsc))

	descending := slices.Clone(ids)
	slices.Reverse(descending)
	require.Equal(t, descending, listPaged(t, sortx.OrderDesc))
}
